package repository

import (
	"context"
	"database/sql"
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
		       t.price_cents, t.stage, t.offer_cents, t.offer_expires_at, t.driver_partner_id, d.name,
		       t.driver_cents, t.paid_at, t.extra_wait_cents, t.cancel_fee_cents, t.flight_cancelled,
		       t.referred_by_partner_id, t.created_by, t.created_at, t.updated_at
		FROM transfers t
		LEFT JOIN partners d ON d.id = t.driver_partner_id`

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
	queryDriverOK = `SELECT EXISTS (SELECT 1 FROM partners WHERE id = $1 AND is_driver AND active)`

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

// Offer abre ou reabre a oferta enquanto ninguém aceitou (RN-02.17, RN-02.22, RN-02.30).
func (r *TransferRepository) Offer(ctx context.Context, id, userID string, offerCents int64, expiresAt time.Time) (transfer.Transfer, error) {
	return r.mutate(ctx, id, userID, func(_ context.Context, _ *sql.Tx, t transfer.Transfer) (string, []any, string, string, error) {
		if t.Stage != transfer.StageWaiting {
			return "", nil, "", "", transfer.ErrWrongStage
		}
		if t.OfferCents != nil && offerCents < *t.OfferCents {
			return "", nil, "", "", transfer.ErrOfferDecreased
		}
		action := "oferta"
		if t.OfferCents != nil {
			action = "oferta_reaberta"
		}
		return `UPDATE transfers SET offer_cents = $2, offer_expires_at = $3, updated_at = now() WHERE id = $1`,
			[]any{offerCents, expiresAt}, action, fmt.Sprintf("%d centavos até %s", offerCents, expiresAt.In(timezone.Location).Format("02/01 15:04")), nil
	})
}

// Accept registra o primeiro motorista que aceitou, pelo valor da oferta (RN-02.17, RN-03.9).
func (r *TransferRepository) Accept(ctx context.Context, id, userID, driverID string) (transfer.Transfer, error) {
	return r.mutate(ctx, id, userID, func(ctx context.Context, tx *sql.Tx, t transfer.Transfer) (string, []any, string, string, error) {
		if t.Stage != transfer.StageWaiting {
			return "", nil, "", "", transfer.ErrWrongStage
		}
		if t.OfferCents == nil {
			return "", nil, "", "", transfer.ErrNoOffer
		}
		var ok bool
		if err := tx.QueryRowContext(ctx, queryDriverOK, driverID).Scan(&ok); err != nil {
			return "", nil, "", "", mapDriverError(err)
		}
		if !ok {
			return "", nil, "", "", transfer.ErrDriverNotFound
		}
		return `UPDATE transfers SET stage = 'com_motorista', driver_partner_id = $2, driver_cents = offer_cents,
			        offer_cents = NULL, offer_expires_at = NULL, updated_at = now() WHERE id = $1`,
			[]any{driverID}, "aceito", "", nil
	})
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
		return `UPDATE transfers SET stage = 'aguardando_motorista', driver_partner_id = NULL, driver_cents = NULL,
			        updated_at = now() WHERE id = $1`, nil, "desistiu", detail, nil
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
			        offer_cents = NULL, offer_expires_at = NULL, updated_at = now() WHERE id = $1`,
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
	var offerExpires, paid sql.NullTime
	var driverID, driverName, referred, createdBy sql.NullString
	err := s.Scan(&t.ID, &t.ClientName, &t.PickupAt, &t.Origin, &t.Destination, &t.Flight, &t.Passengers, &t.Notes,
		&t.PriceCents, &t.Stage, &offer, &offerExpires, &driverID, &driverName,
		&driverCents, &paid, &t.ExtraWaitCents, &t.CancelFeeCents, &t.FlightCancelled,
		&referred, &createdBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	t.PickupAt = t.PickupAt.In(timezone.Location)
	t.OfferCents = nullInt64(offer)
	t.OfferExpiresAt = localTime(offerExpires)
	t.DriverPartnerID = nullString(driverID)
	t.DriverName = nullString(driverName)
	t.DriverCents = nullInt64(driverCents)
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
