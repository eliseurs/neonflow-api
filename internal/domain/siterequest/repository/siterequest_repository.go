package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/account"
	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/domain/siterequest"
)

const (
	invalidTextRepresentation = "22P02"
	uniqueViolation           = "23505"

	// Antes da confirmação do e-mail não há cadastro de cliente: o nome vem da conta.
	requestColumns = `r.id, r.kind, r.status, coalesce(r.customer_id::text, ''), coalesce(c.name, u.name), u.email, r.whatsapp, r.notes, r.referred_by_partner_id,
		r.vehicle_id, coalesce(v.name, ''), r.start_at, r.end_at, r.place, r.km_plan,
		r.days, r.daily_rate_cents, r.total_cents, r.deposit_cents, r.km_limit_per_day, r.extra_km_cents,
		r.origin, r.destination, r.pickup_at, r.passengers, r.flight, r.central_note, r.booking_id, r.transfer_id,
		r.created_at, r.updated_at`
	requestFrom = `
		FROM site_requests r
		LEFT JOIN customers c ON c.id = r.customer_id
		JOIN users u ON u.id = r.user_id
		LEFT JOIN vehicles v ON v.id = r.vehicle_id`

	queryAccount = `
		SELECT id, coalesce(customer_id::text, ''), name FROM users
		WHERE id = $1 AND role = 'cliente' AND active AND email_verified_at IS NOT NULL`
	queryVehicleTerms = `
		SELECT name, daily_rate_cents, in_maintenance, deposit_cents, km_limit_per_day, extra_km_cents, coalesce(min_days, 1)
		FROM vehicles WHERE id = $1`
	// Mesmas etapas que o bloqueio da reserva (EXCLUDE de bookings_no_overlap).
	queryPeriodTaken = `
		SELECT EXISTS (
			SELECT 1 FROM bookings
			WHERE vehicle_id = $1 AND stage NOT IN ('encerrada', 'cancelada')
			  AND tstzrange(start_at, end_at) && tstzrange($2, $3))`
	// O código de indicação vira o parceiro ativo com aquele código; código desconhecido fica de fora.
	queryCreate = `
		INSERT INTO site_requests (kind, customer_id, user_id, whatsapp, notes, referred_by_partner_id,
			vehicle_id, start_at, end_at, place, km_plan, days, daily_rate_cents, total_cents, deposit_cents,
			km_limit_per_day, extra_km_cents, origin, destination, pickup_at, passengers, flight, status)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, (SELECT id FROM partners WHERE referral_code = NULLIF($6, '') AND active),
			NULLIF($7, '')::uuid, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		RETURNING id`
	queryResubmit = `
		UPDATE site_requests SET status = 'solicitado', whatsapp = $2, notes = $3,
			vehicle_id = NULLIF($4, '')::uuid, start_at = $5, end_at = $6, place = $7, km_plan = $8, days = $9,
			daily_rate_cents = $10, total_cents = $11, deposit_cents = $12, km_limit_per_day = $13, extra_km_cents = $14,
			origin = $15, destination = $16, pickup_at = $17, passengers = $18, flight = $19, updated_at = now()
		WHERE id = $1 AND status = 'ajuste_pedido'`
	queryList = `SELECT ` + requestColumns + requestFrom + `
		WHERE (r.status = $1 OR ($1 = '' AND r.status <> 'aguardando_email')) AND ($2 = '' OR r.customer_id::text = $2)
		ORDER BY r.created_at DESC
		LIMIT 500`
	queryFind   = `SELECT ` + requestColumns + requestFrom + ` WHERE r.id = $1`
	queryEvents = `
		SELECT from_status, to_status, note, user_id, created_at FROM site_request_events
		WHERE request_id = $1 ORDER BY created_at, id`
	queryFindGuest = `SELECT id, name, role, active FROM users WHERE lower(email) = $1`
	// Conta sem senha (password_hash vazio não confere com nenhuma senha) e sem confirmar: nasce do pedido.
	queryCreateGuest = `INSERT INTO users (name, email, password_hash, role) VALUES ($1, $2, '', 'cliente') RETURNING id, name`
	querySaveToken   = `
		INSERT INTO account_tokens (purpose, token_hash, user_id, expires_at)
		VALUES ($1, $2, $3, $4)`
	queryEvent = `
		INSERT INTO site_request_events (request_id, from_status, to_status, note, user_id)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5)`
	queryClaim    = `UPDATE site_requests SET status = 'confirmado', updated_at = now() WHERE id = $1 AND status = 'solicitado'`
	queryRelease  = `UPDATE site_requests SET status = 'solicitado', updated_at = now() WHERE id = $1 AND status = 'confirmado' AND booking_id IS NULL AND transfer_id IS NULL`
	queryComplete = `
		UPDATE site_requests SET booking_id = NULLIF($2, '')::uuid, transfer_id = NULLIF($3, '')::uuid, updated_at = now()
		WHERE id = $1 AND status = 'confirmado' AND booking_id IS NULL AND transfer_id IS NULL`
)

type scanner interface {
	Scan(dest ...any) error
}

type SiteRequestRepository struct {
	db *sql.DB
}

func NewSiteRequestRepository(db *sql.DB) *SiteRequestRepository {
	return &SiteRequestRepository{db: db}
}

func (r *SiteRequestRepository) Account(ctx context.Context, userID string) (siterequest.Account, error) {
	var a siterequest.Account
	err := r.db.QueryRowContext(ctx, queryAccount, userID).Scan(&a.UserID, &a.CustomerID, &a.Name)
	if errors.Is(err, sql.ErrNoRows) || isBadID(err) || (err == nil && a.CustomerID == "") {
		return a, siterequest.ErrNoAccount
	}
	if err != nil {
		return a, fmt.Errorf("account: %w", err)
	}
	return a, nil
}

func (r *SiteRequestRepository) VehicleTerms(ctx context.Context, vehicleID string) (siterequest.VehicleTerms, error) {
	var t siterequest.VehicleTerms
	var deposit, extraKm sql.NullInt64
	var kmLimit sql.NullInt32
	err := r.db.QueryRowContext(ctx, queryVehicleTerms, vehicleID).
		Scan(&t.Name, &t.DailyRateCents, &t.InMaintenance, &deposit, &kmLimit, &extraKm, &t.MinDays)
	if errors.Is(err, sql.ErrNoRows) || isBadID(err) {
		return t, booking.ErrVehicleNotFound
	}
	if err != nil {
		return t, fmt.Errorf("vehicle terms: %w", err)
	}
	t.DepositCents, t.ExtraKmCents, t.KmLimitPerDay = i64(deposit), i64(extraKm), i32(kmLimit)
	return t, nil
}

func (r *SiteRequestRepository) PeriodTaken(ctx context.Context, vehicleID string, p booking.Period) (bool, error) {
	var taken bool
	if err := r.db.QueryRowContext(ctx, queryPeriodTaken, vehicleID, p.Start, p.End).Scan(&taken); err != nil {
		return false, fmt.Errorf("period taken: %w", err)
	}
	return taken, nil
}

func (r *SiteRequestRepository) Create(ctx context.Context, n siterequest.NewRequest) (siterequest.Request, error) {
	var id string
	status := n.Status
	if status == "" {
		status = siterequest.StatusRequested
	}
	err := r.inTx(ctx, func(tx *sql.Tx) error {
		args := append([]any{n.Kind, n.CustomerID, n.UserID, n.WhatsApp, n.Notes, n.ReferralCode}, fields(n)...)
		if err := tx.QueryRowContext(ctx, queryCreate, append(args, status)...).Scan(&id); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, queryEvent, id, "", status, "", n.UserID)
		return err
	})
	if err != nil {
		return siterequest.Request{}, fmt.Errorf("create site request: %w", err)
	}
	return r.request(ctx, id)
}

func (r *SiteRequestRepository) Guest(ctx context.Context, name, email string) (siterequest.Guest, error) {
	for attempt := 0; attempt < 2; attempt++ {
		var g siterequest.Guest
		var role string
		var active bool
		err := r.db.QueryRowContext(ctx, queryFindGuest, email).Scan(&g.UserID, &g.Name, &role, &active)
		if err == nil {
			if role != "cliente" || !active {
				return g, siterequest.ErrEmailInUse
			}
			return g, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return g, fmt.Errorf("find guest: %w", err)
		}
		err = r.db.QueryRowContext(ctx, queryCreateGuest, name, email).Scan(&g.UserID, &g.Name)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			continue // outro pedido criou a conta ao mesmo tempo: usa a dele
		}
		if err != nil {
			return g, fmt.Errorf("create guest: %w", err)
		}
		return g, nil
	}
	return siterequest.Guest{}, siterequest.ErrEmailInUse
}

func (r *SiteRequestRepository) SaveToken(ctx context.Context, t account.Token) error {
	if _, err := r.db.ExecContext(ctx, querySaveToken, t.Purpose, t.Hash, t.UserID, t.ExpiresAt); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

func (r *SiteRequestRepository) Resubmit(ctx context.Context, id string, n siterequest.NewRequest) (siterequest.Request, error) {
	err := r.inTx(ctx, func(tx *sql.Tx) error {
		args := append([]any{id, n.WhatsApp, n.Notes}, fields(n)...)
		res, err := tx.ExecContext(ctx, queryResubmit, args...)
		if err != nil {
			return err
		}
		if k, _ := res.RowsAffected(); k == 0 {
			return siterequest.ErrStatusChanged
		}
		_, err = tx.ExecContext(ctx, queryEvent, id, siterequest.StatusAdjustment, siterequest.StatusRequested, "", n.UserID)
		return err
	})
	if err != nil {
		return siterequest.Request{}, wrap(err, "resubmit site request")
	}
	return r.request(ctx, id)
}

// fields são as colunas de aluguel e de transfer, na ordem das queries; as do outro tipo vão nulas.
func fields(n siterequest.NewRequest) []any {
	var start, end, pickup, passengers any
	if n.Kind == siterequest.KindRental {
		start, end = n.Period.Start, n.Period.End
	} else {
		pickup, passengers = n.PickupAt, n.Passengers
	}
	q := n.Quote
	return []any{n.VehicleID, start, end, n.Place, n.KmPlan, q.Days, q.DailyRateCents, q.TotalCents, q.DepositCents,
		q.KmLimitPerDay, q.ExtraKmCents, n.Origin, n.Destination, pickup, passengers, n.Flight}
}

func (r *SiteRequestRepository) List(ctx context.Context, status, customerID string) ([]siterequest.Request, error) {
	rows, err := r.db.QueryContext(ctx, queryList, status, customerID)
	if err != nil {
		return nil, fmt.Errorf("list site requests: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	list := []siterequest.Request{}
	for rows.Next() {
		req, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan site request: %w", err)
		}
		list = append(list, req)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list site requests: %w", err)
	}
	return list, nil
}

func (r *SiteRequestRepository) Find(ctx context.Context, id string) (siterequest.Detail, error) {
	req, err := r.request(ctx, id)
	if err != nil {
		return siterequest.Detail{}, err
	}
	rows, err := r.db.QueryContext(ctx, queryEvents, id)
	if err != nil {
		return siterequest.Detail{}, fmt.Errorf("site request events: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	d := siterequest.Detail{Request: req, Events: []siterequest.Event{}}
	for rows.Next() {
		var e siterequest.Event
		var from sql.NullString
		if err := rows.Scan(&from, &e.ToStatus, &e.Note, &e.UserID, &e.CreatedAt); err != nil {
			return siterequest.Detail{}, fmt.Errorf("scan site request event: %w", err)
		}
		if from.Valid {
			e.FromStatus = &from.String
		}
		d.Events = append(d.Events, e)
	}
	return d, rows.Err()
}

func (r *SiteRequestRepository) request(ctx context.Context, id string) (siterequest.Request, error) {
	req, err := scan(r.db.QueryRowContext(ctx, queryFind, id))
	if errors.Is(err, sql.ErrNoRows) || isBadID(err) {
		return req, siterequest.ErrNotFound
	}
	if err != nil {
		return req, fmt.Errorf("find site request: %w", err)
	}
	return req, nil
}

func (r *SiteRequestRepository) Transition(ctx context.Context, id string, from []string, to, note, userID string) (siterequest.Request, error) {
	err := r.inTx(ctx, func(tx *sql.Tx) error {
		var previous string
		err := tx.QueryRowContext(ctx, `SELECT status FROM site_requests WHERE id = $1 FOR UPDATE`, id).Scan(&previous)
		if errors.Is(err, sql.ErrNoRows) || isBadID(err) {
			return siterequest.ErrNotFound
		}
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `UPDATE site_requests SET status = $3, central_note = CASE WHEN $4 = '' THEN central_note ELSE $4 END,
			updated_at = now() WHERE id = $1 AND status = ANY($2)`, id, from, to, note)
		if err != nil {
			return err
		}
		if k, _ := res.RowsAffected(); k == 0 {
			return siterequest.ErrStatusChanged
		}
		_, err = tx.ExecContext(ctx, queryEvent, id, previous, to, note, userID)
		return err
	})
	if err != nil {
		return siterequest.Request{}, wrap(err, "site request transition")
	}
	return r.request(ctx, id)
}

func (r *SiteRequestRepository) Claim(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, queryClaim, id)
	if err != nil {
		return fmt.Errorf("claim site request: %w", err)
	}
	if k, _ := res.RowsAffected(); k == 0 {
		return siterequest.ErrStatusChanged
	}
	return nil
}

func (r *SiteRequestRepository) Release(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, queryRelease, id); err != nil {
		return fmt.Errorf("release site request: %w", err)
	}
	return nil
}

func (r *SiteRequestRepository) Complete(ctx context.Context, id, bookingID, transferID, userID string) (siterequest.Request, error) {
	err := r.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, queryComplete, id, bookingID, transferID)
		if err != nil {
			return err
		}
		if k, _ := res.RowsAffected(); k == 0 {
			return siterequest.ErrStatusChanged
		}
		_, err = tx.ExecContext(ctx, queryEvent, id, siterequest.StatusRequested, siterequest.StatusConfirmed, "", userID)
		return err
	})
	if err != nil {
		return siterequest.Request{}, wrap(err, "complete site request")
	}
	return r.request(ctx, id)
}

func (r *SiteRequestRepository) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func scan(s scanner) (siterequest.Request, error) {
	var req siterequest.Request
	var referred, vehicleID, bookingID, transferID sql.NullString
	var start, end, pickup sql.NullTime
	var days, kmLimit, passengers sql.NullInt32
	var rate, total, deposit, extraKm sql.NullInt64
	err := s.Scan(&req.ID, &req.Kind, &req.Status, &req.CustomerID, &req.CustomerName, &req.Email, &req.WhatsApp, &req.Notes, &referred,
		&vehicleID, &req.VehicleName, &start, &end, &req.Place, &req.KmPlan,
		&days, &rate, &total, &deposit, &kmLimit, &extraKm,
		&req.Origin, &req.Destination, &pickup, &passengers, &req.Flight, &req.CentralNote, &bookingID, &transferID,
		&req.CreatedAt, &req.UpdatedAt)
	req.ReferredBy, req.VehicleID, req.BookingID, req.TransferID = str(referred), str(vehicleID), str(bookingID), str(transferID)
	req.StartAt, req.EndAt, req.PickupAt = tm(start), tm(end), tm(pickup)
	req.Days, req.KmLimitPerDay, req.Passengers = i32(days), i32(kmLimit), i32(passengers)
	req.DailyRateCents, req.TotalCents, req.DepositCents, req.ExtraKmCents = i64(rate), i64(total), i64(deposit), i64(extraKm)
	return req, err
}

func str(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func tm(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	return &v.Time
}

func i32(v sql.NullInt32) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}

func i64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func isBadID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation
}

// wrap mantém os erros de domínio como estão e dá contexto aos outros.
func wrap(err error, op string) error {
	if errors.Is(err, siterequest.ErrStatusChanged) || errors.Is(err, siterequest.ErrNotFound) {
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}
