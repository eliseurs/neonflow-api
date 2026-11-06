package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/domain/fleet"
	"neonflow-api/internal/pkg/timezone"
)

const (
	exclusionViolation        = "23P01"
	foreignKeyViolation       = "23503"
	invalidTextRepresentation = "22P02"

	bookingColumns = `id, vehicle_id, client_name, start_at, end_at, place, stage, days,
				daily_rate_cents, total_cents, checks, delivered_at, returned_at, extra_minutes, extra_cents,
				deposit_cents, km_limit_per_day, extra_km_cents, delivered_km, returned_km, excess_km, excess_km_cents,
				referred_by_partner_id, paid_at, expires_at, created_by, created_at, updated_at`

	// queryExpire cancela pré-reservas não pagas no prazo e registra a transição (RN-02.11).
	// Roda antes de toda operação, então a disponibilidade nunca considera uma pré-reserva vencida.
	queryExpire = `
		WITH expired AS (
			UPDATE bookings
			SET stage = 'cancelada', updated_at = now()
			WHERE stage = 'preparacao'
			  AND expires_at <= now()
			  AND NOT coalesce((checks->>'payment')::boolean, false)
			RETURNING id
		)
		INSERT INTO booking_events (booking_id, from_stage, to_stage)
		SELECT id, 'preparacao', 'cancelada' FROM expired
	`
	queryList = `
		SELECT ` + bookingColumns + `
		FROM bookings
		WHERE ($1 = '' OR vehicle_id::text = $1)
		ORDER BY start_at, created_at
	`
	queryFindByID = `
		SELECT ` + bookingColumns + `
		FROM bookings
		WHERE id = $1
	`
	queryEvents = `
		SELECT from_stage, to_stage, user_id, created_at
		FROM booking_events
		WHERE booking_id = $1
		ORDER BY created_at, id
	`
	queryLockVehicle = `
				SELECT daily_rate_cents, in_maintenance, deposit_cents, km_limit_per_day, extra_km_cents
		FROM vehicles
		WHERE id = $1
		FOR SHARE
	`
	// A diária do veículo é copiada para a reserva: mudar a tarifa depois não altera o total (RN-02.6).
	queryCreate = `
				INSERT INTO bookings (vehicle_id, client_name, start_at, end_at, place, days,
		                      daily_rate_cents, total_cents, expires_at, created_by,
		                      		                      deposit_cents, km_limit_per_day, extra_km_cents, referred_by_partner_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now() + make_interval(secs => $9), NULLIF($10, '')::uuid,
		        $11, $12, $13, NULLIF($14, '')::uuid)
		RETURNING ` + bookingColumns
	querySetChecks = `
		UPDATE bookings
				SET checks = checks || $3::jsonb,
		    -- Data do pagamento (PEN-11): marcar grava a primeira vez; desmarcar apaga.
		    paid_at = CASE WHEN NOT ($3::jsonb ? 'payment') THEN paid_at
		                   WHEN ($3::jsonb->>'payment')::boolean THEN coalesce(paid_at, now())
		                   ELSE NULL END,
		    updated_at = now()
		WHERE id = $1 AND stage = $2
		RETURNING ` + bookingColumns
	// Pagamentos a parceiros no encerramento (RN-03.12, RN-03.13), com o percentual do parceiro no momento:
	// comissão do indicador sobre as diárias; repasse ao dono do carro parceiro sobre diárias + atraso + km.
	queryCreatePayouts = `
		INSERT INTO partner_payouts (booking_id, partner_id, kind, base_cents, percent_bp, amount_cents)
		SELECT b.id, p.id, 'comissao', b.total_cents, p.commission_bp,
		       CASE WHEN p.commission_bp IS NULL THEN NULL ELSE (b.total_cents * p.commission_bp + 5000) / 10000 END
		FROM bookings b JOIN partners p ON p.id = b.referred_by_partner_id
		WHERE b.id = $1
		UNION ALL
		SELECT b.id, p.id, 'repasse', b.total_cents + b.extra_cents + b.excess_km_cents, p.owner_share_bp,
		       CASE WHEN p.owner_share_bp IS NULL THEN NULL
		            ELSE ((b.total_cents + b.extra_cents + b.excess_km_cents) * p.owner_share_bp + 5000) / 10000 END
		FROM bookings b JOIN vehicles v ON v.id = b.vehicle_id JOIN partners p ON p.id = v.owner_partner_id
		WHERE b.id = $1
		ON CONFLICT (booking_id, kind) DO NOTHING
	`
	// Campos nulos de Record mantêm o valor atual.
	queryTransition = `
		UPDATE bookings
		SET stage = $3,
		    delivered_at = coalesce($4, delivered_at),
		    returned_at = coalesce($5, returned_at),
		    extra_minutes = coalesce($6, extra_minutes),
		    		    extra_cents = coalesce($7, extra_cents),
		    delivered_km = coalesce($8, delivered_km),
		    returned_km = coalesce($9, returned_km),
		    excess_km = coalesce($10, excess_km),
		    excess_km_cents = coalesce($11, excess_km_cents),
		    updated_at = now()
		WHERE id = $1 AND stage = $2
		RETURNING ` + bookingColumns
	// queryVehicleBusy: outra reserva do mesmo veículo que impede a entrega. "em_locacao": o carro
	// ainda está com outro cliente; "devolvida": aguarda o checklist (RN-01.6). Vazio se estiver livre.
	queryVehicleBusy = `
		SELECT coalesce(max(other.stage) FILTER (WHERE other.stage = 'em_locacao'),
		                max(other.stage) FILTER (WHERE other.stage = 'devolvida'), '')
		FROM bookings other
		JOIN bookings b ON b.vehicle_id = other.vehicle_id
		WHERE b.id = $1 AND other.id <> b.id AND other.stage IN ('em_locacao', 'devolvida')
	`
	queryReturnChecklist = `
		SELECT key FROM checklist_items
		WHERE active
		ORDER BY position, label
	`
	queryInsertEvent = `
		INSERT INTO booking_events (booking_id, from_stage, to_stage, user_id)
		VALUES ($1, NULLIF($2, ''), $3, NULLIF($4, '')::uuid)
	`
	queryAvailable = `
				SELECT v.id, v.plate, v.name, v.category, v.owner, v.seats, v.gear,
		       v.daily_rate_cents, v.in_maintenance, v.deposit_cents, v.km_limit_per_day, v.extra_km_cents,
		       v.created_at, v.updated_at
		FROM vehicles v
		WHERE NOT v.in_maintenance
		  AND NOT EXISTS (
			SELECT 1 FROM bookings b
			WHERE b.vehicle_id = v.id
			  AND b.stage NOT IN ('encerrada', 'cancelada')
			  AND tstzrange(b.start_at, b.end_at) && tstzrange($1::timestamptz, $2::timestamptz)
		  )
		ORDER BY v.category, v.name, v.plate
	`
)

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type scanner interface {
	Scan(dest ...any) error
}

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) List(ctx context.Context, vehicleID string) ([]booking.Booking, error) {
	if err := expire(ctx, r.db); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, queryList, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []booking.Booking{}
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	return list, nil
}

func (r *BookingRepository) FindByID(ctx context.Context, id string) (booking.Booking, error) {
	if err := expire(ctx, r.db); err != nil {
		return booking.Booking{}, err
	}
	b, err := scanBooking(r.db.QueryRowContext(ctx, queryFindByID, id))
	return b, mapError(err, booking.ErrBookingNotFound, "find booking")
}

func (r *BookingRepository) Events(ctx context.Context, id string) ([]booking.Event, error) {
	if _, err := r.FindByID(ctx, id); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, queryEvents, id)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	events := []booking.Event{}
	for rows.Next() {
		var e booking.Event
		var from, user sql.NullString
		if err := rows.Scan(&from, &e.ToStage, &user, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.FromStage = nullable(from)
		e.UserID = nullable(user)
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return events, nil
}

// Create bloqueia o período de forma atômica: o EXCLUDE do banco recusa a sobreposição (RN-02.1, RN-02.2).
func (r *BookingRepository) Create(ctx context.Context, n booking.NewBooking) (booking.Booking, error) {
	var created booking.Booking
	err := r.inTx(ctx, func(tx *sql.Tx) error {
		if err := expire(ctx, tx); err != nil {
			return err
		}

		var rate int64
		var inMaintenance bool
		var deposit, extraKm sql.NullInt64
		var kmLimit sql.NullInt32
		err := tx.QueryRowContext(ctx, queryLockVehicle, n.VehicleID).Scan(&rate, &inMaintenance, &deposit, &kmLimit, &extraKm)
		if err := mapError(err, booking.ErrVehicleNotFound, "lock vehicle"); err != nil {
			return err
		}
		if inMaintenance {
			return booking.ErrVehicleInMaintenance // RN-01.1
		}
		// RN-02.18: o plano vem da escolha da central; sem escolha, limitado se o veículo tiver limite.
		if n.KmPlan == booking.KmPlanLimitado && !kmLimit.Valid {
			return booking.ErrKmPlanUnavailable
		}
		if n.KmPlan == booking.KmPlanLivre {
			kmLimit, extraKm = sql.NullInt32{}, sql.NullInt64{}
		}

		days := n.Period.Days()
		created, err = scanBooking(tx.QueryRowContext(ctx, queryCreate,
			n.VehicleID, n.ClientName, n.Period.Start, n.Period.End, n.Place, days,
			rate, int64(days)*rate, booking.PaymentWindow.Seconds(), n.CreatedBy, deposit, kmLimit, extraKm, n.ReferredBy))
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
				return booking.ErrReferrerNotFound
			}
			return mapError(err, booking.ErrBookingNotFound, "create booking")
		}
		return insertEvent(ctx, tx, created.ID, "", created.Stage, n.CreatedBy)
	})
	return created, err
}

// SetChecks grava as conferências só se a reserva ainda estiver na etapa informada.
func (r *BookingRepository) SetChecks(ctx context.Context, id, stage string, checks map[string]bool) (booking.Booking, error) {
	if err := expire(ctx, r.db); err != nil {
		return booking.Booking{}, err
	}
	payload, err := json.Marshal(checks)
	if err != nil {
		return booking.Booking{}, fmt.Errorf("marshal checks: %w", err)
	}
	b, err := scanBooking(r.db.QueryRowContext(ctx, querySetChecks, id, stage, string(payload)))
	return b, mapError(err, booking.ErrStageChanged, "set checks")
}

// Transition muda a etapa só se ela ainda for "from", então dois cliques não avançam duas vezes (RN-02.4).
func (r *BookingRepository) Transition(ctx context.Context, id, from, to, userID string, rec booking.Record) (booking.Booking, error) {
	var updated booking.Booking
	err := r.inTx(ctx, func(tx *sql.Tx) error {
		if err := expire(ctx, tx); err != nil {
			return err
		}
		// O veículo só sai se não estiver com outro cliente e, depois de uma devolução, com o
		// checklist aprovado (RN-01.6).
		if to == booking.StageEmLocacao {
			var busy string
			if err := tx.QueryRowContext(ctx, queryVehicleBusy, id).Scan(&busy); err != nil {
				return mapError(err, booking.ErrBookingNotFound, "check vehicle busy")
			}
			switch busy {
			case booking.StageEmLocacao:
				return booking.ErrVehicleInUse
			case booking.StageDevolvida:
				return booking.ErrAwaitingChecklist
			}
		}
		var err error
		updated, err = scanBooking(tx.QueryRowContext(ctx, queryTransition, id, from, to,
			rec.DeliveredAt, rec.ReturnedAt, rec.ExtraMinutes, rec.ExtraCents,
			rec.DeliveredKm, rec.ReturnedKm, rec.ExcessKm, rec.ExcessKmCents))
		if err != nil {
			return mapError(err, booking.ErrStageChanged, "transition booking")
		}
		// Reserva encerrada (paga e finalizada) gera a comissão do indicador e o repasse ao dono (RN-03.12, RN-03.13).
		if to == booking.StageEncerrada {
			if _, err := tx.ExecContext(ctx, queryCreatePayouts, id); err != nil {
				return fmt.Errorf("create payouts: %w", err)
			}
		}
		return insertEvent(ctx, tx, id, from, to, userID)
	})
	return updated, err
}

func (r *BookingRepository) Available(ctx context.Context, p booking.Period) ([]fleet.Vehicle, error) {
	if err := expire(ctx, r.db); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, queryAvailable, p.Start, p.End)
	if err != nil {
		return nil, fmt.Errorf("available vehicles: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	vehicles := []fleet.Vehicle{}
	for rows.Next() {
		var v fleet.Vehicle
		var deposit, extraKm sql.NullInt64
		var kmLimit sql.NullInt32
		if err := rows.Scan(&v.ID, &v.Plate, &v.Name, &v.Category, &v.Owner, &v.Seats, &v.Gear,
			&v.DailyRateCents, &v.InMaintenance, &deposit, &kmLimit, &extraKm, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		v.DepositCents, v.ExtraKmCents, v.KmLimitPerDay = nullInt64(deposit), nullInt64(extraKm), nullInt(kmLimit)
		vehicles = append(vehicles, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("available vehicles: %w", err)
	}
	return vehicles, nil
}

func (r *BookingRepository) ReturnChecklist(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, queryReturnChecklist)
	if err != nil {
		return nil, fmt.Errorf("return checklist: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan checklist key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("return checklist: %w", err)
	}
	return keys, nil
}

func (r *BookingRepository) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return mapError(err, booking.ErrBookingNotFound, "commit")
	}
	return nil
}

func expire(ctx context.Context, db execer) error {
	if _, err := db.ExecContext(ctx, queryExpire); err != nil {
		return fmt.Errorf("expire bookings: %w", err)
	}
	return nil
}

func insertEvent(ctx context.Context, db execer, id, from, to, userID string) error {
	if _, err := db.ExecContext(ctx, queryInsertEvent, id, from, to, userID); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

func scanBooking(s scanner) (booking.Booking, error) {
	var b booking.Booking
	var start, end time.Time
	var checks []byte
	var createdBy sql.NullString
	var delivered, returned, paid sql.NullTime
	var deposit, extraKm sql.NullInt64
	var kmLimit, deliveredKm, returnedKm sql.NullInt32
	var referredBy sql.NullString
	err := s.Scan(&b.ID, &b.VehicleID, &b.ClientName, &start, &end, &b.Place, &b.Stage, &b.Days,
		&b.DailyRateCents, &b.TotalCents, &checks, &delivered, &returned, &b.ExtraMinutes, &b.ExtraCents,
		&deposit, &kmLimit, &extraKm, &deliveredKm, &returnedKm, &b.ExcessKm, &b.ExcessKmCents,
		&referredBy, &paid, &b.ExpiresAt, &createdBy, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return b, err
	}
	b.StartAt = start.In(timezone.Location)
	b.EndAt = end.In(timezone.Location)
	b.StartDate = timezone.Date(start)
	b.EndDate = timezone.Date(end)
	b.CreatedBy = nullable(createdBy)
	b.ReferredByPartnerID = nullable(referredBy)
	b.DeliveredAt = localTime(delivered)
	b.ReturnedAt = localTime(returned)
	b.PaidAt = localTime(paid)
	b.DepositCents = nullInt64(deposit)
	b.ExtraKmCents = nullInt64(extraKm)
	b.KmLimitPerDay = nullInt(kmLimit)
	b.DeliveredKm = nullInt(deliveredKm)
	b.ReturnedKm = nullInt(returnedKm)
	b.Checks = map[string]bool{}
	if err := json.Unmarshal(checks, &b.Checks); err != nil {
		return b, fmt.Errorf("decode checks: %w", err)
	}
	return b, nil
}

func nullInt64(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}

func nullInt(n sql.NullInt32) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int32)
	return &v
}

func localTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	local := t.Time.In(timezone.Location)
	return &local
}

func nullable(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

// mapError traduz linha inexistente e id mal formatado em notFound e sobreposição em ErrPeriodUnavailable.
func mapError(err, notFound error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return notFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case exclusionViolation:
			return booking.ErrPeriodUnavailable
		case invalidTextRepresentation:
			return notFound
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
