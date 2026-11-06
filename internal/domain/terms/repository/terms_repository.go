package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/terms"
	"neonflow-api/internal/pkg/timezone"
)

const (
	invalidTextRepresentation = "22P02"

	queryVersions = `SELECT id, version, body, created_at FROM terms_versions ORDER BY version DESC`
	queryCurrent  = `SELECT id, version, body, created_at FROM terms_versions ORDER BY version DESC LIMIT 1`
	queryPublish  = `
		INSERT INTO terms_versions (version, body, created_by)
		SELECT coalesce(max(version), 0) + 1, $1, NULLIF($2, '')::uuid FROM terms_versions
		RETURNING id, version, body, created_at`
	queryBookingStage = `SELECT stage FROM bookings WHERE id = $1`
	queryCreateLink   = `INSERT INTO terms_links (booking_id, token_hash, expires_at, created_by) VALUES ($1, $2, $3, NULLIF($4, '')::uuid)`
	queryLink         = `
		SELECT l.id, l.booking_id, b.stage, l.expires_at
		FROM terms_links l JOIN bookings b ON b.id = l.booking_id
		WHERE l.token_hash = $1 AND l.expires_at > now()`
	queryPublic = `
		SELECT b.client_name, v.name, b.start_at, b.end_at, b.total_cents
		FROM bookings b JOIN vehicles v ON v.id = b.vehicle_id WHERE b.id = $1`
	queryLast = `
		SELECT tv.version, a.typed_name, a.accepted_at
		FROM terms_acceptances a JOIN terms_versions tv ON tv.id = a.version_id
		WHERE a.booking_id = $1 ORDER BY a.accepted_at DESC LIMIT 1`
	queryAccept = `
		INSERT INTO terms_acceptances (booking_id, version_id, link_id, typed_name)
		VALUES ($1, $2, $3, $4) RETURNING accepted_at`
	queryMarkTerms = `UPDATE bookings SET checks = checks || '{"terms": true}'::jsonb, updated_at = now() WHERE id = $1`
)

type TermsRepository struct {
	db *sql.DB
}

func NewTermsRepository(db *sql.DB) *TermsRepository {
	return &TermsRepository{db: db}
}

func (r *TermsRepository) Versions(ctx context.Context) ([]terms.Version, error) {
	rows, err := r.db.QueryContext(ctx, queryVersions)
	if err != nil {
		return nil, fmt.Errorf("list terms: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []terms.Version{}
	for rows.Next() {
		var v terms.Version
		if err := rows.Scan(&v.ID, &v.Version, &v.Body, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan terms: %w", err)
		}
		list = append(list, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list terms: %w", err)
	}
	return list, nil
}

func (r *TermsRepository) Current(ctx context.Context) (terms.Version, error) {
	var v terms.Version
	err := r.db.QueryRowContext(ctx, queryCurrent).Scan(&v.ID, &v.Version, &v.Body, &v.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return v, terms.ErrNoTerms
	}
	if err != nil {
		return v, fmt.Errorf("current terms: %w", err)
	}
	return v, nil
}

func (r *TermsRepository) CreateVersion(ctx context.Context, body, userID string) (terms.Version, error) {
	var v terms.Version
	if err := r.db.QueryRowContext(ctx, queryPublish, body, userID).Scan(&v.ID, &v.Version, &v.Body, &v.CreatedAt); err != nil {
		return v, fmt.Errorf("publish terms: %w", err)
	}
	return v, nil
}

func (r *TermsRepository) BookingStage(ctx context.Context, bookingID string) (string, error) {
	var stage string
	err := r.db.QueryRowContext(ctx, queryBookingStage, bookingID).Scan(&stage)
	return stage, mapNotFound(err, terms.ErrBookingNotFound, "booking stage")
}

func (r *TermsRepository) CreateLink(ctx context.Context, bookingID, tokenHash string, expiresAt time.Time, userID string) error {
	if _, err := r.db.ExecContext(ctx, queryCreateLink, bookingID, tokenHash, expiresAt, userID); err != nil {
		return fmt.Errorf("create terms link: %w", err)
	}
	return nil
}

func (r *TermsRepository) LinkByHash(ctx context.Context, hash string) (terms.LinkInfo, error) {
	var l terms.LinkInfo
	err := r.db.QueryRowContext(ctx, queryLink, hash).Scan(&l.ID, &l.BookingID, &l.BookingStage, &l.ExpiresAt)
	return l, mapNotFound(err, terms.ErrLinkNotFound, "terms link")
}

func (r *TermsRepository) Public(ctx context.Context, link terms.LinkInfo) (terms.Public, error) {
	p := terms.Public{ExpiresAt: link.ExpiresAt.In(timezone.Location)}
	err := r.db.QueryRowContext(ctx, queryPublic, link.BookingID).Scan(&p.ClientName, &p.VehicleName, &p.StartAt, &p.EndAt, &p.TotalCents)
	if err != nil {
		return p, mapNotFound(err, terms.ErrLinkNotFound, "terms booking")
	}
	p.StartAt, p.EndAt = p.StartAt.In(timezone.Location), p.EndAt.In(timezone.Location)
	if p.Terms, err = r.Current(ctx); err != nil {
		return p, err
	}
	if p.Accepted, err = r.LastAcceptance(ctx, link.BookingID); err != nil {
		return p, err
	}
	return p, nil
}

func (r *TermsRepository) Accept(ctx context.Context, link terms.LinkInfo, version int, typedName string) (terms.Acceptance, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return terms.Acceptance{}, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// A versão aceita tem de ser a que o cliente leu; trava para não mudar no meio.
	var currentID string
	var current int
	err = tx.QueryRowContext(ctx, `SELECT id, version FROM terms_versions ORDER BY version DESC LIMIT 1 FOR SHARE`).Scan(&currentID, &current)
	if errors.Is(err, sql.ErrNoRows) {
		return terms.Acceptance{}, terms.ErrNoTerms
	}
	if err != nil {
		return terms.Acceptance{}, fmt.Errorf("current terms: %w", err)
	}
	if current != version {
		return terms.Acceptance{}, terms.ErrVersionChanged
	}
	a := terms.Acceptance{Version: version, TypedName: typedName}
	if err := tx.QueryRowContext(ctx, queryAccept, link.BookingID, currentID, link.ID, typedName).Scan(&a.AcceptedAt); err != nil {
		return terms.Acceptance{}, fmt.Errorf("accept terms: %w", err)
	}
	if _, err := tx.ExecContext(ctx, queryMarkTerms, link.BookingID); err != nil {
		return terms.Acceptance{}, fmt.Errorf("mark terms: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return terms.Acceptance{}, fmt.Errorf("commit: %w", err)
	}
	a.AcceptedAt = a.AcceptedAt.In(timezone.Location)
	return a, nil
}

func (r *TermsRepository) LastAcceptance(ctx context.Context, bookingID string) (*terms.Acceptance, error) {
	var a terms.Acceptance
	err := r.db.QueryRowContext(ctx, queryLast, bookingID).Scan(&a.Version, &a.TypedName, &a.AcceptedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		if mapped := mapNotFound(err, terms.ErrBookingNotFound, "last acceptance"); errors.Is(mapped, terms.ErrBookingNotFound) {
			return nil, mapped
		}
		return nil, fmt.Errorf("last acceptance: %w", err)
	}
	a.AcceptedAt = a.AcceptedAt.In(timezone.Location)
	return &a, nil
}

func mapNotFound(err error, notFound error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return notFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return notFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
