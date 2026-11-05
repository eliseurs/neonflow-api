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
)

const (
	exclusionViolation        = "23P01"
	invalidTextRepresentation = "22P02"

	bookingColumns = `id, vehicle_id, client_name, start_date, end_date, place, stage, days,
		daily_rate_cents, total_cents, checks, expires_at, created_by, created_at, updated_at`

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
		ORDER BY start_date, created_at
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
		SELECT daily_rate_cents, in_maintenance
		FROM vehicles
		WHERE id = $1
		FOR SHARE
	`
	// A diária do veículo é copiada para a reserva: mudar a tarifa depois não altera o total (RN-02.6).
	queryCreate = `
		INSERT INTO bookings (vehicle_id, client_name, start_date, end_date, place, days,
		                      daily_rate_cents, total_cents, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now() + make_interval(secs => $9), NULLIF($10, '')::uuid)
		RETURNING ` + bookingColumns
	querySetChecks = `
		UPDATE bookings
		SET checks = checks || $3::jsonb, updated_at = now()
		WHERE id = $1 AND stage = $2
		RETURNING ` + bookingColumns
	queryTransition = `
		UPDATE bookings
		SET stage = $3, updated_at = now()
		WHERE id = $1 AND stage = $2
		RETURNING ` + bookingColumns
	queryInsertEvent = `
		INSERT INTO booking_events (booking_id, from_stage, to_stage, user_id)
		VALUES ($1, NULLIF($2, ''), $3, NULLIF($4, '')::uuid)
	`
	queryAvailable = `
		SELECT v.id, v.plate, v.name, v.category, v.owner, v.seats, v.gear,
		       v.daily_rate_cents, v.in_maintenance, v.created_at, v.updated_at
		FROM vehicles v
		WHERE NOT v.in_maintenance
		  AND NOT EXISTS (
			SELECT 1 FROM bookings b
			WHERE b.vehicle_id = v.id
			  AND b.stage NOT IN ('encerrada', 'cancelada')
			  AND daterange(b.start_date, b.end_date) && daterange($1::date, $2::date)
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
		err := tx.QueryRowContext(ctx, queryLockVehicle, n.VehicleID).Scan(&rate, &inMaintenance)
		if err := mapError(err, booking.ErrVehicleNotFound, "lock vehicle"); err != nil {
			return err
		}
		if inMaintenance {
			return booking.ErrVehicleInMaintenance // RN-01.1
		}

		days := n.Period.Days()
		created, err = scanBooking(tx.QueryRowContext(ctx, queryCreate,
			n.VehicleID, n.ClientName, n.Period.Start, n.Period.End, n.Place, days,
			rate, int64(days)*rate, booking.PaymentWindow.Seconds(), n.CreatedBy))
		if err != nil {
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
func (r *BookingRepository) Transition(ctx context.Context, id, from, to, userID string) (booking.Booking, error) {
	var updated booking.Booking
	err := r.inTx(ctx, func(tx *sql.Tx) error {
		if err := expire(ctx, tx); err != nil {
			return err
		}
		var err error
		updated, err = scanBooking(tx.QueryRowContext(ctx, queryTransition, id, from, to))
		if err != nil {
			return mapError(err, booking.ErrStageChanged, "transition booking")
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
		if err := rows.Scan(&v.ID, &v.Plate, &v.Name, &v.Category, &v.Owner, &v.Seats, &v.Gear,
			&v.DailyRateCents, &v.InMaintenance, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		vehicles = append(vehicles, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("available vehicles: %w", err)
	}
	return vehicles, nil
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
	err := s.Scan(&b.ID, &b.VehicleID, &b.ClientName, &start, &end, &b.Place, &b.Stage, &b.Days,
		&b.DailyRateCents, &b.TotalCents, &checks, &b.ExpiresAt, &createdBy, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return b, err
	}
	b.StartDate = start.Format(booking.DateLayout)
	b.EndDate = end.Format(booking.DateLayout)
	b.CreatedBy = nullable(createdBy)
	b.Checks = map[string]bool{}
	if err := json.Unmarshal(checks, &b.Checks); err != nil {
		return b, fmt.Errorf("decode checks: %w", err)
	}
	return b, nil
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
