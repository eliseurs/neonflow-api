package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/customer"
	"neonflow-api/internal/pkg/timezone"
)

const (
	uniqueViolation           = "23505"
	invalidTextRepresentation = "22P02"

	customerColumns = `c.id, c.name, c.whatsapp, c.email, c.language, c.notes, c.anonymized_at, c.created_at, c.updated_at,
		(SELECT count(*) FROM bookings b WHERE b.customer_id = c.id),
		(SELECT count(*) FROM transfers t WHERE t.customer_id = c.id)`

	// Busca por trecho do nome ou do e-mail, ou pelos dígitos do WhatsApp.
	queryList = `
		SELECT ` + customerColumns + `
		FROM customers c
		WHERE $1 = ''
		   OR lower(c.name) LIKE '%' || $1 || '%'
		   OR lower(c.email) LIKE '%' || $1 || '%'
		   OR ($2 <> '' AND c.whatsapp LIKE '%' || $2 || '%')
		ORDER BY c.anonymized_at IS NOT NULL, c.updated_at DESC
		LIMIT 200`
	queryFind   = `SELECT ` + customerColumns + ` FROM customers c WHERE c.id = $1`
	queryOrders = `
		SELECT 'reserva', b.id, b.start_at, v.name || ' · ' || b.days || CASE WHEN b.days = 1 THEN ' diária' ELSE ' diárias' END,
		       b.stage, b.total_cents + b.extra_cents + b.excess_km_cents
		FROM bookings b JOIN vehicles v ON v.id = b.vehicle_id
		WHERE b.customer_id = $1
		UNION ALL
		SELECT 'transfer', t.id, t.pickup_at, t.origin || ' → ' || t.destination, t.stage, t.price_cents + t.extra_wait_cents
		FROM transfers t
		WHERE t.customer_id = $1
		ORDER BY 3 DESC`
	queryCreate = `
		INSERT INTO customers (name, whatsapp, email, language, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	queryUpdate = `
		UPDATE customers SET name = $2, whatsapp = $3, email = $4, language = $5, notes = $6, updated_at = now()
		WHERE id = $1 AND anonymized_at IS NULL`
	// Anonimiza o cadastro e o nome gravado nos pedidos; valores e datas ficam (registro financeiro).
	queryAnonymize = `
		WITH c AS (
			UPDATE customers SET name = $2, whatsapp = '', email = '', notes = '', anonymized_at = now(), updated_at = now()
			WHERE id = $1 AND anonymized_at IS NULL
			RETURNING id
		), b AS (
			UPDATE bookings SET client_name = $2 WHERE customer_id IN (SELECT id FROM c)
		), t AS (
			UPDATE transfers SET client_name = $2, notes = '' WHERE customer_id IN (SELECT id FROM c)
		)
		SELECT count(*) FROM c`
)

type scanner interface {
	Scan(dest ...any) error
}

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) List(ctx context.Context, text, digits string) ([]customer.Customer, error) {
	rows, err := r.db.QueryContext(ctx, queryList, text, digits)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []customer.Customer{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	return list, nil
}

func (r *CustomerRepository) find(ctx context.Context, id string) (customer.Customer, error) {
	c, err := scan(r.db.QueryRowContext(ctx, queryFind, id))
	return c, mapError(err, "find customer")
}

func (r *CustomerRepository) Get(ctx context.Context, id string) (customer.Detail, error) {
	c, err := r.find(ctx, id)
	if err != nil {
		return customer.Detail{}, err
	}
	rows, err := r.db.QueryContext(ctx, queryOrders, id)
	if err != nil {
		return customer.Detail{}, fmt.Errorf("customer orders: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	d := customer.Detail{Customer: c, Orders: []customer.Order{}}
	for rows.Next() {
		var o customer.Order
		if err := rows.Scan(&o.Kind, &o.ID, &o.At, &o.Summary, &o.Stage, &o.TotalCents); err != nil {
			return customer.Detail{}, fmt.Errorf("scan order: %w", err)
		}
		o.At = o.At.In(timezone.Location)
		d.Orders = append(d.Orders, o)
	}
	if err := rows.Err(); err != nil {
		return customer.Detail{}, fmt.Errorf("customer orders: %w", err)
	}
	return d, nil
}

func (r *CustomerRepository) Create(ctx context.Context, req customer.Request) (customer.Customer, error) {
	var id string
	err := r.db.QueryRowContext(ctx, queryCreate, req.Name, req.WhatsApp, req.Email, req.Language, req.Notes).Scan(&id)
	if err != nil {
		return customer.Customer{}, mapError(err, "create customer")
	}
	return r.find(ctx, id)
}

func (r *CustomerRepository) Update(ctx context.Context, id string, req customer.Request) (customer.Customer, error) {
	res, err := r.db.ExecContext(ctx, queryUpdate, id, req.Name, req.WhatsApp, req.Email, req.Language, req.Notes)
	if err != nil {
		return customer.Customer{}, mapError(err, "update customer")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return r.notChanged(ctx, id)
	}
	return r.find(ctx, id)
}

func (r *CustomerRepository) Anonymize(ctx context.Context, id string) (customer.Customer, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, queryAnonymize, id, customer.AnonymizedName).Scan(&n); err != nil {
		return customer.Customer{}, mapError(err, "anonymize customer")
	}
	if n == 0 {
		return r.notChanged(ctx, id)
	}
	return r.find(ctx, id)
}

// notChanged distingue cliente inexistente de cliente já anonimizado.
func (r *CustomerRepository) notChanged(ctx context.Context, id string) (customer.Customer, error) {
	if _, err := r.find(ctx, id); err != nil {
		return customer.Customer{}, err
	}
	return customer.Customer{}, customer.ErrAnonymized
}

func scan(s scanner) (customer.Customer, error) {
	var c customer.Customer
	var anonymized sql.NullTime
	err := s.Scan(&c.ID, &c.Name, &c.WhatsApp, &c.Email, &c.Language, &c.Notes, &anonymized, &c.CreatedAt, &c.UpdatedAt,
		&c.BookingCount, &c.TransferCount)
	if err != nil {
		return c, err
	}
	if anonymized.Valid {
		t := anonymized.Time.In(timezone.Location)
		c.AnonymizedAt = &t
	}
	return c, nil
}

func mapError(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return customer.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			return customer.ErrWhatsAppTaken
		case invalidTextRepresentation:
			return customer.ErrNotFound
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
