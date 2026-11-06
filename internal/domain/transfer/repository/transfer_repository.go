package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/transfer"
	"neonflow-api/internal/pkg/timezone"
)

const (
	foreignKeyViolation       = "23503"
	invalidTextRepresentation = "22P02"

	transferSelect = `
		SELECT t.id, t.client_name, t.pickup_at, t.origin, t.destination, t.flight, t.passengers, t.notes,
		       t.price_cents, t.stage, t.offer_cents, t.driver_partner_id, d.name,
		       t.driver_cents, t.vehicle_id, v.name, t.paid_at, t.extra_wait_cents, t.cancel_fee_cents, t.flight_cancelled,
		       t.referred_by_partner_id, t.created_by, t.created_at, t.updated_at
		FROM transfers t
		LEFT JOIN partners d ON d.id = t.driver_partner_id
		LEFT JOIN vehicles v ON v.id = t.vehicle_id`

	queryList   = transferSelect + ` ORDER BY t.pickup_at DESC`
	queryFind   = transferSelect + ` WHERE t.id = $1`
	queryLock   = transferSelect + ` WHERE t.id = $1 FOR UPDATE OF t`
	queryCreate = `
		INSERT INTO transfers (client_name, pickup_at, origin, destination, flight, passengers, notes, price_cents,
		                       referred_by_partner_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, '')::uuid, NULLIF($10, '')::uuid)
		RETURNING id`
	queryEvent  = `INSERT INTO transfer_events (transfer_id, action, detail, user_id) VALUES ($1, $2, $3, NULLIF($4, '')::uuid)`
	queryEvents = `
		SELECT e.action, e.detail, u.name, e.created_at
		FROM transfer_events e LEFT JOIN users u ON u.id = e.user_id
		WHERE e.transfer_id = $1 ORDER BY e.created_at, e.id`
	// Pode dirigir: parceiro ativo motorista ou proprietário (RN-03.10 revista).
	queryDriverName = `SELECT name FROM partners WHERE id = $1 AND active AND (is_driver OR is_owner)`
	queryVehicleOK  = `SELECT EXISTS (SELECT 1 FROM vehicles WHERE id = $1)`
	queryDrivers    = `
		SELECT p.id, p.name, p.whatsapp, p.is_driver, p.is_owner,
		       coalesce((SELECT json_agg(o.pickup_at ORDER BY o.pickup_at) FROM transfers o
		                 WHERE o.driver_partner_id = p.id AND o.id <> $1
		                   AND o.stage IN ('com_motorista', 'em_andamento')
		                   AND (o.pickup_at AT TIME ZONE 'America/Sao_Paulo')::date = ($2::timestamptz AT TIME ZONE 'America/Sao_Paulo')::date),
		                '[]'),
		       coalesce((SELECT json_agg(json_build_object('id', v.id, 'name', v.name, 'plate', v.plate) ORDER BY v.name)
		                 FROM vehicles v WHERE v.owner_partner_id = p.id), '[]')
		FROM partners p
		WHERE p.active AND (p.is_driver OR p.is_owner)
		ORDER BY p.is_driver DESC, p.name`

	// Na conclusão: o motorista recebe oferta + espera (RN-03.9, RN-02.16) e o indicador, o percentual
	// sobre o preço, saindo da parte do Neon Flow (RN-03.14).
	queryPayouts = `
		INSERT INTO partner_payouts (transfer_id, partner_id, kind, base_cents, percent_bp, amount_cents)
		SELECT t.id, t.driver_partner_id, 'motorista', t.driver_cents + t.extra_wait_cents, NULL,
		       t.driver_cents + t.extra_wait_cents
		FROM transfers t WHERE t.id = $1
		UNION ALL
		SELECT t.id, p.id, 'comissao', t.price_cents, p.commission_bp,
		       CASE WHEN p.commission_bp IS NULL THEN NULL ELSE (t.price_cents * p.commission_bp + 5000) / 10000 END
		FROM transfers t JOIN partners p ON p.id = t.referred_by_partner_id WHERE t.id = $1
		ON CONFLICT (transfer_id, kind) DO NOTHING`
)

type scanner interface {
	Scan(dest ...any) error
}

type TransferRepository struct {
	db *sql.DB
}

func NewTransferRepository(db *sql.DB) *TransferRepository {
	return &TransferRepository{db: db}
}

func (r *TransferRepository) List(ctx context.Context) ([]transfer.Transfer, error) {
	rows, err := r.db.QueryContext(ctx, queryList)
	if err != nil {
		return nil, fmt.Errorf("list transfers: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []transfer.Transfer{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transfer: %w", err)
		}
		list = append(list, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list transfers: %w", err)
	}
	return list, nil
}

func (r *TransferRepository) FindByID(ctx context.Context, id string) (transfer.Transfer, error) {
	t, err := scan(r.db.QueryRowContext(ctx, queryFind, id))
	return t, mapError(err, "find transfer")
}

func (r *TransferRepository) Events(ctx context.Context, id string) ([]transfer.Event, error) {
	if _, err := r.FindByID(ctx, id); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, queryEvents, id)
	if err != nil {
		return nil, fmt.Errorf("list transfer events: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	events := []transfer.Event{}
	for rows.Next() {
		var e transfer.Event
		var user sql.NullString
		if err := rows.Scan(&e.Action, &e.Detail, &user, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan transfer event: %w", err)
		}
		e.CreatedAt = e.CreatedAt.In(timezone.Location)
		if user.Valid {
			e.UserName = &user.String
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list transfer events: %w", err)
	}
	return events, nil
}

func (r *TransferRepository) Create(ctx context.Context, n transfer.NewTransfer) (transfer.Transfer, error) {
	var created transfer.Transfer
	err := r.inTx(ctx, func(tx *sql.Tx) error {
		var id string
		err := tx.QueryRowContext(ctx, queryCreate, n.ClientName, n.PickupAt, n.Origin, n.Destination, n.Flight,
			n.Passengers, n.Notes, n.PriceCents, n.ReferredBy, n.CreatedBy).Scan(&id)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
				return transfer.ErrReferrerNotFound
			}
			return fmt.Errorf("create transfer: %w", err)
		}
		if _, err := tx.ExecContext(ctx, queryEvent, id, "criado", "", n.CreatedBy); err != nil {
			return fmt.Errorf("transfer event: %w", err)
		}
		created, err = scan(tx.QueryRowContext(ctx, queryFind, id))
		return err
	})
	return created, err
}

// step é uma ação sobre a corrida travada: confere o estado atual e devolve a atualização e o evento.
type step func(ctx context.Context, tx *sql.Tx, t transfer.Transfer) (update string, args []any, action, detail string, err error)

func (r *TransferRepository) mutate(ctx context.Context, id, userID string, s step) (transfer.Transfer, error) {
	var out transfer.Transfer
	err := r.inTx(ctx, func(tx *sql.Tx) error {
		t, err := scan(tx.QueryRowContext(ctx, queryLock, id))
		if err != nil {
			return mapError(err, "lock transfer")
		}
		update, args, action, detail, err := s(ctx, tx, t)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, update, append([]any{id}, args...)...); err != nil {
			return fmt.Errorf("%s transfer: %w", action, err)
		}
		if _, err := tx.ExecContext(ctx, queryEvent, id, action, detail, userID); err != nil {
			return fmt.Errorf("transfer event: %w", err)
		}
		// Concluída: pagamento do motorista e comissão do indicador na mesma transação (RN-03.8).
		if action == actionDone {
			if _, err := tx.ExecContext(ctx, queryPayouts, id); err != nil {
				return fmt.Errorf("transfer payouts: %w", err)
			}
		}
		out, err = scan(tx.QueryRowContext(ctx, queryFind, id))
		return err
	})
	return out, err
}

const actionDone = "concluido"

// Drivers lista quem pode fazer a corrida, com as outras corridas do mesmo dia e os carros de cada um.
func (r *TransferRepository) Drivers(ctx context.Context, id string) ([]transfer.Driver, error) {
	t, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, queryDrivers, id, t.PickupAt)
	if err != nil {
		return nil, fmt.Errorf("list drivers: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []transfer.Driver{}
	for rows.Next() {
		var d transfer.Driver
		var isDriver, isOwner bool
		var sameDay, vehicles []byte
		if err := rows.Scan(&d.PartnerID, &d.Name, &d.WhatsApp, &isDriver, &isOwner, &sameDay, &vehicles); err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		d.Kinds = []string{}
		if isDriver {
			d.Kinds = append(d.Kinds, "motorista")
		}
		if isOwner {
			d.Kinds = append(d.Kinds, "proprietario")
		}
		if err := json.Unmarshal(sameDay, &d.SameDay); err != nil {
			return nil, fmt.Errorf("decode same day: %w", err)
		}
		for i := range d.SameDay {
			d.SameDay[i] = d.SameDay[i].In(timezone.Location)
		}
		if err := json.Unmarshal(vehicles, &d.Vehicles); err != nil {
			return nil, fmt.Errorf("decode vehicles: %w", err)
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list drivers: %w", err)
	}
	return list, nil
}

// Offer define ou aumenta o valor para o motorista enquanto ninguém aceitou (RN-02.22).
func (r *TransferRepository) Offer(ctx context.Context, id, userID string, offerCents int64) (transfer.Transfer, error) {
	return r.mutate(ctx, id, userID, func(_ context.Context, _ *sql.Tx, t transfer.Transfer) (string, []any, string, string, error) {
		if t.Stage != transfer.StageWaiting {
			return "", nil, "", "", transfer.ErrWrongStage
		}
		if err := transfer.ValidateOffer(offerCents, t.OfferCents); err != nil {
			return "", nil, "", "", err
		}
		action, detail := "valor", transfer.FormatReais(offerCents)
		if t.OfferCents != nil && offerCents > *t.OfferCents {
			action, detail = "valor_aumentado", transfer.FormatReais(*t.OfferCents)+" → "+detail
		}
		return `UPDATE transfers SET offer_cents = $2, updated_at = now() WHERE id = $1`, []any{offerCents}, action, detail, nil
	})
}

// Notify registra que a central chamou o motorista pelo WhatsApp (RN-02.27).
func (r *TransferRepository) Notify(ctx context.Context, id, userID, driverID string) (transfer.Transfer, error) {
	return r.mutate(ctx, id, userID, func(ctx context.Context, tx *sql.Tx, t transfer.Transfer) (string, []any, string, string, error) {
		if t.Stage != transfer.StageWaiting {
			return "", nil, "", "", transfer.ErrWrongStage
		}
		name, err := driverName(ctx, tx, driverID)
		if err != nil {
			return "", nil, "", "", err
		}
		return `UPDATE transfers SET updated_at = now() WHERE id = $1`, nil, "avisado", name, nil
	})
}

// Accept marca o motorista que aceitou, com o valor final e o carro da corrida (RN-02.17, RN-03.14).
func (r *TransferRepository) Accept(ctx context.Context, id, userID string, req transfer.AcceptRequest) (transfer.Transfer, error) {
	return r.mutate(ctx, id, userID, func(ctx context.Context, tx *sql.Tx, t transfer.Transfer) (string, []any, string, string, error) {
		if t.Stage != transfer.StageWaiting {
			return "", nil, "", "", transfer.ErrWrongStage
		}
		valid, err := req.Validate(t.OfferCents)
		if err != nil {
			return "", nil, "", "", err
		}
		value := t.OfferCents
		if valid.OfferCents != nil {
			value = valid.OfferCents
		}
		name, err := driverName(ctx, tx, valid.DriverPartnerID)
		if err != nil {
			return "", nil, "", "", err
		}
		if valid.VehicleID != "" {
			var ok bool
			if err := tx.QueryRowContext(ctx, queryVehicleOK, valid.VehicleID).Scan(&ok); err != nil {
				return "", nil, "", "", fmt.Errorf("check vehicle: %w", err)
			}
			if !ok {
				return "", nil, "", "", transfer.ErrVehicleNotFound
			}
		}
		return `UPDATE transfers SET stage = 'com_motorista', driver_partner_id = $2, driver_cents = $3,
			        vehicle_id = NULLIF($4, '')::uuid, offer_cents = NULL, updated_at = now() WHERE id = $1`,
			[]any{valid.DriverPartnerID, *value, valid.VehicleID}, "aceito", name + " · " + transfer.FormatReais(*value), nil
	})
}

func driverName(ctx context.Context, tx *sql.Tx, driverID string) (string, error) {
	var name string
	err := tx.QueryRowContext(ctx, queryDriverName, driverID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", transfer.ErrDriverNotFound
	}
	if err != nil {
		return "", mapDriverError(err)
	}
	return name, nil
}

// Release: o motorista desistiu; a corrida volta a ser oferecida, sem penalidade (RN-02.31).
func (r *TransferRepository) Release(ctx context.Context, id, userID string) (transfer.Transfer, error) {
	return r.mutate(ctx, id, userID, func(_ context.Context, _ *sql.Tx, t transfer.Transfer) (string, []any, string, string, error) {
		if t.Stage != transfer.StageAssigned {
			return "", nil, "", "", transfer.ErrWrongStage
		}
		detail := ""
		if t.DriverName != nil {
			detail = *t.DriverName
		}
		// O valor volta a valer para o próximo motorista.
		return `UPDATE transfers SET stage = 'aguardando_motorista', offer_cents = driver_cents, driver_partner_id = NULL,
			        driver_cents = NULL, vehicle_id = NULL, updated_at = now() WHERE id = $1`, nil, "desistiu", detail, nil
	})
}

// SetPaid marca ou desmarca o pagamento do cliente, enquanto a corrida não terminou.
func (r *TransferRepository) SetPaid(ctx context.Context, id, userID string, paid bool) (transfer.Transfer, error) {
	return r.mutate(ctx, id, userID, func(_ context.Context, _ *sql.Tx, t transfer.Transfer) (string, []any, string, string, error) {
		if t.Stage == transfer.StageDone || t.Stage == transfer.StageCancelled {
			return "", nil, "", "", transfer.ErrWrongStage
		}
		if paid {
			return `UPDATE transfers SET paid_at = coalesce(paid_at, now()), updated_at = now() WHERE id = $1`, nil, "pago", "", nil
		}
		return `UPDATE transfers SET paid_at = NULL, updated_at = now() WHERE id = $1`, nil, "pagamento_desfeito", "", nil
	})
}

// Start: o motorista saiu para a corrida.
func (r *TransferRepository) Start(ctx context.Context, id, userID string) (transfer.Transfer, error) {
	return r.mutate(ctx, id, userID, func(_ context.Context, _ *sql.Tx, t transfer.Transfer) (string, []any, string, string, error) {
		if t.Stage != transfer.StageAssigned {
			return "", nil, "", "", transfer.ErrWrongStage
		}
		return `UPDATE transfers SET stage = 'em_andamento', updated_at = now() WHERE id = $1`, nil, "iniciado", "", nil
	})
}

// Complete conclui a corrida paga e gera o pagamento do motorista e a comissão do indicador (RN-03.8).
func (r *TransferRepository) Complete(ctx context.Context, id, userID string, extraWaitCents int64) (transfer.Transfer, error) {
	return r.mutate(ctx, id, userID, func(_ context.Context, _ *sql.Tx, t transfer.Transfer) (string, []any, string, string, error) {
		if t.Stage != transfer.StageOnTrip {
			return "", nil, "", "", transfer.ErrWrongStage
		}
		if t.PaidAt == nil {
			return "", nil, "", "", transfer.ErrNotPaid
		}
		return `UPDATE transfers SET stage = 'concluido', extra_wait_cents = $2, updated_at = now() WHERE id = $1`,
			[]any{extraWaitCents}, actionDone, "", nil
	})
}

// Cancel encerra a corrida que ainda não começou, com a multa calculada pelo serviço (RN-02.10).
func (r *TransferRepository) Cancel(ctx context.Context, id, userID string, feeCents int64, flightCancelled bool) (transfer.Transfer, error) {
	return r.mutate(ctx, id, userID, func(_ context.Context, _ *sql.Tx, t transfer.Transfer) (string, []any, string, string, error) {
		if !transfer.Cancellable(t.Stage) {
			return "", nil, "", "", transfer.ErrWrongStage
		}
		detail := ""
		if flightCancelled {
			detail = "voo cancelado pela companhia"
		}
		return `UPDATE transfers SET stage = 'cancelado', cancel_fee_cents = $2, flight_cancelled = $3,
			        offer_cents = NULL, updated_at = now() WHERE id = $1`,
			[]any{feeCents, flightCancelled}, "cancelado", detail, nil
	})
}

func (r *TransferRepository) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func scan(s scanner) (transfer.Transfer, error) {
	var t transfer.Transfer
	var offer, driverCents sql.NullInt64
	var paid sql.NullTime
	var driverID, driverName, vehicleID, vehicleName, referred, createdBy sql.NullString
	err := s.Scan(&t.ID, &t.ClientName, &t.PickupAt, &t.Origin, &t.Destination, &t.Flight, &t.Passengers, &t.Notes,
		&t.PriceCents, &t.Stage, &offer, &driverID, &driverName,
		&driverCents, &vehicleID, &vehicleName, &paid, &t.ExtraWaitCents, &t.CancelFeeCents, &t.FlightCancelled,
		&referred, &createdBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	t.PickupAt = t.PickupAt.In(timezone.Location)
	t.OfferCents = nullInt64(offer)
	t.DriverPartnerID = nullString(driverID)
	t.DriverName = nullString(driverName)
	t.DriverCents = nullInt64(driverCents)
	t.VehicleID = nullString(vehicleID)
	t.VehicleName = nullString(vehicleName)
	t.PaidAt = localTime(paid)
	t.ReferredByPartnerID = nullString(referred)
	t.CreatedBy = nullString(createdBy)
	return t, nil
}

func nullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func nullString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func localTime(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time.In(timezone.Location)
	return &t
}

func mapDriverError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return transfer.ErrDriverNotFound
	}
	return fmt.Errorf("check driver: %w", err)
}

func mapError(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return transfer.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return transfer.ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
