package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/fleet"
	"neonflow-api/internal/pkg/timezone"
)

const (
	uniqueViolation           = "23505"
	invalidTextRepresentation = "22P02"

	// queryVehicles traz o veículo com o estado exibido e a próxima reserva (RN-01.1, RN-01.3, RN-01.6).
	// Conta a reserva que ainda não terminou (devolução combinada depois de agora).
	// A pré-reserva vencida e não paga não conta, mesmo antes de a rotina de expiração cancelá-la (RN-02.11).
	queryVehicles = `
		SELECT v.id, v.plate, v.name, v.category, v.owner, v.seats, v.gear, v.daily_rate_cents,
		       v.in_maintenance, v.deposit_cents, v.km_limit_per_day, v.extra_km_cents, v.created_at, v.updated_at,
		       CASE
		           WHEN v.in_maintenance THEN 'manutencao'
		           WHEN EXISTS (SELECT 1 FROM bookings b WHERE b.vehicle_id = v.id AND b.stage = 'em_locacao') THEN 'em_locacao'
		           WHEN EXISTS (SELECT 1 FROM bookings b WHERE b.vehicle_id = v.id AND b.stage = 'devolvida') THEN 'aguardando_checklist'
		           WHEN nb.id IS NOT NULL THEN 'reservado'
		           ELSE 'livre'
		       END,
		       nb.id, nb.client_name, nb.start_at, nb.end_at, nb.stage
		FROM vehicles v
		LEFT JOIN LATERAL (
		    SELECT b.id, b.client_name, b.start_at, b.end_at, b.stage
		    FROM bookings b
		    WHERE b.vehicle_id = v.id
		      AND (b.stage = 'confirmada'
		           OR (b.stage = 'preparacao'
		               AND (b.expires_at > now() OR coalesce((b.checks->>'payment')::boolean, false))))
		      AND b.end_at > now()
		    ORDER BY b.start_at, b.created_at
		    LIMIT 1
		) nb ON true
	`
	queryList     = queryVehicles + ` ORDER BY v.name, v.plate`
	queryFindByID = queryVehicles + ` WHERE v.id = $1`
	queryCreate   = `
		INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents, in_maintenance,
		                      deposit_cents, km_limit_per_day, extra_km_cents)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`
	queryUpdate = `
		UPDATE vehicles
		SET plate = $2, name = $3, category = $4, owner = $5, seats = $6, gear = $7,
		    daily_rate_cents = $8, in_maintenance = $9, deposit_cents = $10, km_limit_per_day = $11,
		    extra_km_cents = $12, updated_at = now()
		WHERE id = $1
		RETURNING id`
)

type scanner interface {
	Scan(dest ...any) error
}

type FleetRepository struct {
	db *sql.DB
}

func NewFleetRepository(db *sql.DB) *FleetRepository {
	return &FleetRepository{db: db}
}

func (r *FleetRepository) List(ctx context.Context) ([]fleet.Vehicle, error) {
	rows, err := r.db.QueryContext(ctx, queryList)
	if err != nil {
		return nil, fmt.Errorf("list vehicles: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	vehicles := []fleet.Vehicle{}
	for rows.Next() {
		v, err := scanVehicle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		vehicles = append(vehicles, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list vehicles: %w", err)
	}
	return vehicles, nil
}

func (r *FleetRepository) FindByID(ctx context.Context, id string) (fleet.Vehicle, error) {
	v, err := scanVehicle(r.db.QueryRowContext(ctx, queryFindByID, id))
	return v, mapError(err, "find vehicle")
}

// Create e Update devolvem o veículo relido, já com o estado calculado.
func (r *FleetRepository) Create(ctx context.Context, n fleet.VehicleRequest) (fleet.Vehicle, error) {
	var id string
	err := r.db.QueryRowContext(ctx, queryCreate,
		n.Plate, n.Name, n.Category, n.Owner, n.Seats, n.Gear, n.DailyRateCents, n.InMaintenance,
		n.DepositCents, n.KmLimitPerDay, n.ExtraKmCents).Scan(&id)
	if err := mapError(err, "create vehicle"); err != nil {
		return fleet.Vehicle{}, err
	}
	return r.FindByID(ctx, id)
}

func (r *FleetRepository) Update(ctx context.Context, id string, n fleet.VehicleRequest) (fleet.Vehicle, error) {
	err := r.db.QueryRowContext(ctx, queryUpdate,
		id, n.Plate, n.Name, n.Category, n.Owner, n.Seats, n.Gear, n.DailyRateCents, n.InMaintenance,
		n.DepositCents, n.KmLimitPerDay, n.ExtraKmCents).Scan(&id)
	if err := mapError(err, "update vehicle"); err != nil {
		return fleet.Vehicle{}, err
	}
	return r.FindByID(ctx, id)
}

func scanVehicle(s scanner) (fleet.Vehicle, error) {
	var v fleet.Vehicle
	var nbID, nbClient, nbStage sql.NullString
	var nbStart, nbEnd sql.NullTime
	var deposit, extraKm sql.NullInt64
	var kmLimit sql.NullInt32
	err := s.Scan(&v.ID, &v.Plate, &v.Name, &v.Category, &v.Owner, &v.Seats, &v.Gear,
		&v.DailyRateCents, &v.InMaintenance, &deposit, &kmLimit, &extraKm, &v.CreatedAt, &v.UpdatedAt,
		&v.Status, &nbID, &nbClient, &nbStart, &nbEnd, &nbStage)
	if err != nil {
		return v, err
	}
	v.DepositCents = nullInt64(deposit)
	v.ExtraKmCents = nullInt64(extraKm)
	if kmLimit.Valid {
		limit := int(kmLimit.Int32)
		v.KmLimitPerDay = &limit
	}
	if nbID.Valid {
		v.NextBooking = &fleet.NextBooking{
			ID: nbID.String, ClientName: nbClient.String, Stage: nbStage.String,
			StartAt: nbStart.Time.In(timezone.Location), EndAt: nbEnd.Time.In(timezone.Location),
			StartDate: timezone.Date(nbStart.Time), EndDate: timezone.Date(nbEnd.Time),
		}
	}
	return v, nil
}

func nullInt64(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}

// mapError traduz linha inexistente e id mal formatado em ErrVehicleNotFound e placa repetida em ErrPlateTaken.
func mapError(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return fleet.ErrVehicleNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			return fleet.ErrPlateTaken
		case invalidTextRepresentation:
			return fleet.ErrVehicleNotFound
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
