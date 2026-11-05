package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/fleet"
)

const (
	uniqueViolation           = "23505"
	invalidTextRepresentation = "22P02"

	vehicleColumns = `id, plate, name, category, owner, seats, gear, daily_rate_cents, in_maintenance, created_at, updated_at`

	queryList = `
		SELECT ` + vehicleColumns + `
		FROM vehicles
		ORDER BY name, plate
	`
	queryFindByID = `
		SELECT ` + vehicleColumns + `
		FROM vehicles
		WHERE id = $1
	`
	queryCreate = `
		INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents, in_maintenance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING ` + vehicleColumns
	queryUpdate = `
		UPDATE vehicles
		SET plate = $2, name = $3, category = $4, owner = $5, seats = $6, gear = $7,
		    daily_rate_cents = $8, in_maintenance = $9, updated_at = now()
		WHERE id = $1
		RETURNING ` + vehicleColumns
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

func (r *FleetRepository) Create(ctx context.Context, n fleet.VehicleRequest) (fleet.Vehicle, error) {
	v, err := scanVehicle(r.db.QueryRowContext(ctx, queryCreate,
		n.Plate, n.Name, n.Category, n.Owner, n.Seats, n.Gear, n.DailyRateCents, n.InMaintenance))
	return v, mapError(err, "create vehicle")
}

func (r *FleetRepository) Update(ctx context.Context, id string, n fleet.VehicleRequest) (fleet.Vehicle, error) {
	v, err := scanVehicle(r.db.QueryRowContext(ctx, queryUpdate,
		id, n.Plate, n.Name, n.Category, n.Owner, n.Seats, n.Gear, n.DailyRateCents, n.InMaintenance))
	return v, mapError(err, "update vehicle")
}

func scanVehicle(s scanner) (fleet.Vehicle, error) {
	var v fleet.Vehicle
	err := s.Scan(&v.ID, &v.Plate, &v.Name, &v.Category, &v.Owner, &v.Seats, &v.Gear,
		&v.DailyRateCents, &v.InMaintenance, &v.CreatedAt, &v.UpdatedAt)
	return v, err
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
