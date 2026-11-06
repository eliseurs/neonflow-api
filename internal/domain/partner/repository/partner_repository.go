package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/partner"
)

const (
	uniqueViolation           = "23505"
	invalidTextRepresentation = "22P02"

	partnerColumns = `id, name, is_owner, is_driver, is_referrer, whatsapp, email, city, notes, referral_code,
		commission_bp, owner_share_bp, driver_share_bp, active, created_at, updated_at`

	// kind vazio lista todos; ativos primeiro.
	queryList = `
		SELECT ` + partnerColumns + `
		FROM partners
		WHERE $1 = ''
		   OR ($1 = 'proprietario' AND is_owner)
		   OR ($1 = 'motorista' AND is_driver)
		   OR ($1 = 'indicador' AND is_referrer)
		ORDER BY active DESC, name
	`
	queryFindByID   = `SELECT ` + partnerColumns + ` FROM partners WHERE id = $1`
	queryFindByCode = `SELECT ` + partnerColumns + ` FROM partners WHERE referral_code = $1`
	queryCreate     = `
		INSERT INTO partners (name, is_owner, is_driver, is_referrer, whatsapp, email, city, notes, referral_code,
		                      commission_bp, owner_share_bp, driver_share_bp, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, coalesce($13, true))
		RETURNING ` + partnerColumns
	queryUpdate = `
		UPDATE partners
		SET name = $2, is_owner = $3, is_driver = $4, is_referrer = $5, whatsapp = $6, email = $7, city = $8,
		    notes = $9, commission_bp = $10, owner_share_bp = $11, driver_share_bp = $12,
		    active = coalesce($13, active), updated_at = now()
		WHERE id = $1
		RETURNING ` + partnerColumns
)

type scanner interface {
	Scan(dest ...any) error
}

type PartnerRepository struct {
	db *sql.DB
}

func NewPartnerRepository(db *sql.DB) *PartnerRepository {
	return &PartnerRepository{db: db}
}

func (r *PartnerRepository) List(ctx context.Context, kind string) ([]partner.Partner, error) {
	rows, err := r.db.QueryContext(ctx, queryList, kind)
	if err != nil {
		return nil, fmt.Errorf("list partners: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []partner.Partner{}
	for rows.Next() {
		p, err := scanPartner(rows)
		if err != nil {
			return nil, fmt.Errorf("scan partner: %w", err)
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list partners: %w", err)
	}
	return list, nil
}

func (r *PartnerRepository) FindByID(ctx context.Context, id string) (partner.Partner, error) {
	p, err := scanPartner(r.db.QueryRowContext(ctx, queryFindByID, id))
	return p, mapError(err, "find partner")
}

func (r *PartnerRepository) FindByCode(ctx context.Context, code string) (partner.Partner, error) {
	p, err := scanPartner(r.db.QueryRowContext(ctx, queryFindByCode, code))
	return p, mapError(err, "find partner by code")
}

func (r *PartnerRepository) Create(ctx context.Context, code string, req partner.Request) (partner.Partner, error) {
	p, err := scanPartner(r.db.QueryRowContext(ctx, queryCreate, args(req, code)...))
	return p, mapError(err, "create partner")
}

func (r *PartnerRepository) Update(ctx context.Context, id string, req partner.Request) (partner.Partner, error) {
	a := args(req, "")
	p, err := scanPartner(r.db.QueryRowContext(ctx, queryUpdate,
		id, a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7], a[9], a[10], a[11], a[12]))
	return p, mapError(err, "update partner")
}

// args monta os parâmetros na ordem do INSERT (o código fica na posição 8).
func args(req partner.Request, code string) []any {
	var active sql.NullBool
	if req.Active != nil {
		active = sql.NullBool{Bool: *req.Active, Valid: true}
	}
	return []any{
		req.Name, req.Has(partner.KindOwner), req.Has(partner.KindDriver), req.Has(partner.KindReferrer),
		req.WhatsApp, req.Email, req.City, req.Notes, code,
		partner.ToBasisPoints(req.CommissionPercent), partner.ToBasisPoints(req.OwnerSharePercent),
		partner.ToBasisPoints(req.DriverSharePercent), active,
	}
}

func scanPartner(s scanner) (partner.Partner, error) {
	var p partner.Partner
	var owner, driver, referrer bool
	var commission, ownerShare, driverShare sql.NullInt32
	err := s.Scan(&p.ID, &p.Name, &owner, &driver, &referrer, &p.WhatsApp, &p.Email, &p.City, &p.Notes,
		&p.ReferralCode, &commission, &ownerShare, &driverShare, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return p, err
	}
	p.Kinds = []string{}
	for _, k := range []struct {
		on   bool
		kind string
	}{{owner, partner.KindOwner}, {driver, partner.KindDriver}, {referrer, partner.KindReferrer}} {
		if k.on {
			p.Kinds = append(p.Kinds, k.kind)
		}
	}
	p.CommissionPercent = percent(commission)
	p.OwnerSharePercent = percent(ownerShare)
	p.DriverSharePercent = percent(driverShare)
	return p, nil
}

func percent(n sql.NullInt32) *float64 {
	if !n.Valid {
		return nil
	}
	bp := int(n.Int32)
	return partner.FromBasisPoints(&bp)
}

func mapError(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return partner.ErrPartnerNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			return partner.ErrCodeTaken
		case invalidTextRepresentation:
			return partner.ErrPartnerNotFound
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
