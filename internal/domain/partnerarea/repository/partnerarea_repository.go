package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/partnerarea"
)

const (
	invalidTextRepresentation = "22P02"
	// limit: as mais recentes de cada lista; o histórico completo fica com a central.
	limit = 200

	queryPartner = `
		SELECT p.id, p.name, p.is_owner, p.is_driver, p.is_referrer, p.referral_code
		FROM users u JOIN partners p ON p.id = u.partner_id
		WHERE u.id = $1 AND u.active AND p.active`
	queryVehicles = `
		SELECT id, name, plate, in_maintenance FROM vehicles
		WHERE owner_partner_id = $1 ORDER BY name`
	payoutColumns = `pp.base_cents, pp.percent_bp, pp.amount_cents, pp.paid_at`
	// Reservas dos carros do proprietário, sem nome nem contato do cliente (RN-07.9).
	queryBookings = `
		SELECT b.id, v.name, b.start_at, b.end_at, b.stage, b.days, b.extra_minutes, b.excess_km, ` + payoutColumns + `
		FROM bookings b
		JOIN vehicles v ON v.id = b.vehicle_id
		LEFT JOIN partner_payouts pp ON pp.booking_id = b.id AND pp.kind = 'repasse' AND pp.partner_id = $1
		WHERE v.owner_partner_id = $1
		ORDER BY b.start_at DESC LIMIT $2`
	// Corridas do motorista (RN-07.10). O telefone e as observações do cliente ficam com a central.
	queryRides = `
		SELECT t.id, t.pickup_at, t.origin, t.destination, t.flight, t.passengers, t.client_name, t.stage, t.driver_cents, ` + payoutColumns + `
		FROM transfers t
		LEFT JOIN partner_payouts pp ON pp.transfer_id = t.id AND pp.kind = 'motorista' AND pp.partner_id = $1
		WHERE t.driver_partner_id = $1
		ORDER BY t.pickup_at DESC LIMIT $2`
	// Oferta aberta (PEN-74): corridas aguardando motorista com o valor da central, as mais próximas primeiro.
	queryOffers = `
		SELECT id, pickup_at, origin, destination, flight, passengers, client_name, offer_cents
		FROM transfers
		WHERE stage = 'aguardando_motorista' AND offer_cents IS NOT NULL
		ORDER BY pickup_at LIMIT $1`
	queryOwnsVehicle = `SELECT EXISTS (SELECT 1 FROM vehicles WHERE id = $1 AND owner_partner_id = $2)`
	// Indicações (RN-07.11): reservas e transfers com o parceiro em "Indicado por", sem o cliente.
	queryReferrals = `
		SELECT * FROM (
			SELECT b.id, 'aluguel' AS kind, b.start_at AS at, b.stage, ` + payoutColumns + `
			FROM bookings b
			LEFT JOIN partner_payouts pp ON pp.booking_id = b.id AND pp.kind = 'comissao' AND pp.partner_id = $1
			WHERE b.referred_by_partner_id = $1
			UNION ALL
			SELECT t.id, 'transfer', t.pickup_at, t.stage, ` + payoutColumns + `
			FROM transfers t
			LEFT JOIN partner_payouts pp ON pp.transfer_id = t.id AND pp.kind = 'comissao' AND pp.partner_id = $1
			WHERE t.referred_by_partner_id = $1
		) r ORDER BY at DESC LIMIT $2`
)

type scanner interface {
	Scan(dest ...any) error
}

type PartnerAreaRepository struct {
	db *sql.DB
}

func NewPartnerAreaRepository(db *sql.DB) *PartnerAreaRepository {
	return &PartnerAreaRepository{db: db}
}

func (r *PartnerAreaRepository) Partner(ctx context.Context, userID string) (partnerarea.Partner, error) {
	var p partnerarea.Partner
	var owner, driver, referrer bool
	err := r.db.QueryRowContext(ctx, queryPartner, userID).Scan(&p.ID, &p.Name, &owner, &driver, &referrer, &p.ReferralCode)
	var pgErr *pgconn.PgError
	if errors.Is(err, sql.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation) {
		return p, partnerarea.ErrNoPartner
	}
	if err != nil {
		return p, fmt.Errorf("partner area: %w", err)
	}
	p.Kinds = []string{}
	for _, k := range []struct {
		kind string
		has  bool
	}{{"proprietario", owner}, {"motorista", driver}, {"indicador", referrer}} {
		if k.has {
			p.Kinds = append(p.Kinds, k.kind)
		}
	}
	return p, nil
}

func (r *PartnerAreaRepository) Vehicles(ctx context.Context, partnerID string) ([]partnerarea.Vehicle, error) {
	return list(ctx, r.db, "partner vehicles", queryVehicles, []any{partnerID}, func(s scanner) (partnerarea.Vehicle, error) {
		var v partnerarea.Vehicle
		return v, s.Scan(&v.ID, &v.Name, &v.Plate, &v.InMaintenance)
	})
}

func (r *PartnerAreaRepository) Bookings(ctx context.Context, partnerID string) ([]partnerarea.Booking, error) {
	return list(ctx, r.db, "partner bookings", queryBookings, []any{partnerID, limit}, func(s scanner) (partnerarea.Booking, error) {
		var b partnerarea.Booking
		var p payoutRow
		err := s.Scan(append([]any{&b.ID, &b.VehicleName, &b.StartAt, &b.EndAt, &b.Stage, &b.Days, &b.ExtraMinutes, &b.ExcessKm}, p.dest()...)...)
		b.Payout = p.value()
		return b, err
	})
}

func (r *PartnerAreaRepository) Rides(ctx context.Context, partnerID string) ([]partnerarea.Ride, error) {
	return list(ctx, r.db, "partner rides", queryRides, []any{partnerID, limit}, func(s scanner) (partnerarea.Ride, error) {
		var t partnerarea.Ride
		var p payoutRow
		err := s.Scan(append([]any{&t.ID, &t.PickupAt, &t.Origin, &t.Destination, &t.Flight, &t.Passengers, &t.ClientName, &t.Stage, &t.DriverCents}, p.dest()...)...)
		t.Payout = p.value()
		return t, err
	})
}

func (r *PartnerAreaRepository) Referrals(ctx context.Context, partnerID string) ([]partnerarea.Referral, error) {
	return list(ctx, r.db, "partner referrals", queryReferrals, []any{partnerID, limit}, func(s scanner) (partnerarea.Referral, error) {
		var ref partnerarea.Referral
		var p payoutRow
		err := s.Scan(append([]any{&ref.ID, &ref.Kind, &ref.At, &ref.Stage}, p.dest()...)...)
		ref.Payout = p.value()
		return ref, err
	})
}

func (r *PartnerAreaRepository) Offers(ctx context.Context) ([]partnerarea.Offer, error) {
	return list(ctx, r.db, "partner offers", queryOffers, []any{limit}, func(s scanner) (partnerarea.Offer, error) {
		var o partnerarea.Offer
		return o, s.Scan(&o.ID, &o.PickupAt, &o.Origin, &o.Destination, &o.Flight, &o.Passengers, &o.ClientName, &o.OfferCents)
	})
}

func (r *PartnerAreaRepository) OwnsVehicle(ctx context.Context, partnerID, vehicleID string) (bool, error) {
	var owns bool
	err := r.db.QueryRowContext(ctx, queryOwnsVehicle, vehicleID, partnerID).Scan(&owns)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("owns vehicle: %w", err)
	}
	return owns, nil
}

// payoutRow lê as colunas do pagamento, que vêm nulas enquanto ele não foi gerado.
type payoutRow struct {
	base, amount sql.NullInt64
	percent      sql.NullInt32
	paid         sql.NullTime
}

func (p *payoutRow) dest() []any { return []any{&p.base, &p.percent, &p.amount, &p.paid} }

func (p *payoutRow) value() *partnerarea.Payout {
	if !p.base.Valid {
		return nil
	}
	out := &partnerarea.Payout{BaseCents: p.base.Int64}
	if p.percent.Valid {
		v := int(p.percent.Int32)
		out.PercentBP = &v
	}
	if p.amount.Valid {
		out.AmountCents = &p.amount.Int64
	}
	if p.paid.Valid {
		out.PaidAt = &p.paid.Time
	}
	return out
}

func list[T any](ctx context.Context, db *sql.DB, op, query string, args []any, scan func(scanner) (T, error)) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close() //nolint:errcheck
	out := []T{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return out, nil
}
