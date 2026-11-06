package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/payout"
	"neonflow-api/internal/pkg/timezone"
)

const (
	invalidTextRepresentation = "22P02"

	payoutSelect = `
		SELECT o.id, o.kind, o.booking_id, b.client_name, v.name, b.end_at, o.partner_id, p.name,
		       o.base_cents, o.percent_bp, o.amount_cents, o.paid_at, u.name, o.created_at
		FROM partner_payouts o
		JOIN bookings b ON b.id = o.booking_id
		JOIN vehicles v ON v.id = b.vehicle_id
		JOIN partners p ON p.id = o.partner_id
		LEFT JOIN users u ON u.id = o.paid_by`

	// Pendentes primeiro; dentro de cada grupo, as mais recentes no topo.
	queryList = payoutSelect + `
		WHERE ($1 = '' OR o.kind = $1)
		  AND ($2 = '' OR ($2 = 'pendente' AND o.paid_at IS NULL) OR ($2 = 'paga' AND o.paid_at IS NOT NULL))
		  AND ($3 = '' OR o.partner_id::text = $3)
		ORDER BY o.paid_at IS NOT NULL, coalesce(o.paid_at, o.created_at) DESC`
	queryFind = payoutSelect + ` WHERE o.id = $1`

	// Percentual atual do parceiro conforme o tipo: comissão ou repasse ao dono.
	queryLock = `
		SELECT o.paid_at IS NOT NULL, o.percent_bp,
		       CASE o.kind WHEN 'comissao' THEN p.commission_bp ELSE p.owner_share_bp END, o.base_cents
		FROM partner_payouts o JOIN partners p ON p.id = o.partner_id
		WHERE o.id = $1 FOR UPDATE OF o`
	queryPay = `UPDATE partner_payouts SET percent_bp = $2, amount_cents = $3, paid_at = now(), paid_by = $4 WHERE id = $1`
)

type scanner interface {
	Scan(dest ...any) error
}

type PayoutRepository struct {
	db *sql.DB
}

func NewPayoutRepository(db *sql.DB) *PayoutRepository {
	return &PayoutRepository{db: db}
}

func (r *PayoutRepository) List(ctx context.Context, f payout.Filter) ([]payout.Payout, error) {
	rows, err := r.db.QueryContext(ctx, queryList, f.Kind, f.Status, f.PartnerID)
	if err != nil {
		return nil, fmt.Errorf("list payouts: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []payout.Payout{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payout: %w", err)
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list payouts: %w", err)
	}
	return list, nil
}

func (r *PayoutRepository) MarkPaid(ctx context.Context, id, userID string) (payout.Payout, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return payout.Payout{}, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var paid bool
	var percent, partnerPercent sql.NullInt32
	var base int64
	if err := tx.QueryRowContext(ctx, queryLock, id).Scan(&paid, &percent, &partnerPercent, &base); err != nil {
		return payout.Payout{}, mapError(err, "lock payout")
	}
	if paid {
		return payout.Payout{}, payout.ErrAlreadyPaid
	}
	// Percentual a definir no encerramento: vale o do parceiro agora.
	if !percent.Valid {
		percent = partnerPercent
	}
	if !percent.Valid {
		return payout.Payout{}, payout.ErrPercentUndefined
	}
	amount := payout.Amount(base, int(percent.Int32))
	if _, err := tx.ExecContext(ctx, queryPay, id, percent.Int32, amount, userID); err != nil {
		return payout.Payout{}, fmt.Errorf("pay payout: %w", err)
	}
	p, err := scan(tx.QueryRowContext(ctx, queryFind, id))
	if err != nil {
		return payout.Payout{}, mapError(err, "find payout")
	}
	if err := tx.Commit(); err != nil {
		return payout.Payout{}, fmt.Errorf("commit: %w", err)
	}
	return p, nil
}

func scan(s scanner) (payout.Payout, error) {
	var p payout.Payout
	var percent sql.NullInt32
	var amount sql.NullInt64
	var paidAt sql.NullTime
	var paidBy sql.NullString
	err := s.Scan(&p.ID, &p.Kind, &p.BookingID, &p.ClientName, &p.VehicleName, &p.EndAt, &p.PartnerID, &p.PartnerName,
		&p.BaseCents, &percent, &amount, &paidAt, &paidBy, &p.CreatedAt)
	if err != nil {
		return p, err
	}
	p.EndAt = p.EndAt.In(timezone.Location)
	if percent.Valid {
		v := float64(percent.Int32) / 100
		p.Percent = &v
	}
	if amount.Valid {
		p.AmountCents = &amount.Int64
	}
	p.Status = payout.StatusPending
	if paidAt.Valid {
		t := paidAt.Time.In(timezone.Location)
		p.PaidAt, p.Status = &t, payout.StatusPaid
	}
	if paidBy.Valid {
		p.PaidByName = &paidBy.String
	}
	return p, nil
}

func mapError(err error, op string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return payout.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return payout.ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
