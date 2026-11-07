package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/maintenance"
)

const (
	foreignKeyViolation       = "23503"
	invalidTextRepresentation = "22P02"

	columns = `id, vehicle_id, to_char(date, 'YYYY-MM-DD'), kind, description, cost_cents, odometer_km, workshop, created_at`

	queryVehicle = `SELECT EXISTS (SELECT 1 FROM vehicles WHERE id = $1)`
	queryList    = `SELECT ` + columns + ` FROM vehicle_maintenance WHERE vehicle_id = $1 ORDER BY date DESC, created_at DESC`
	queryCreate  = `
		INSERT INTO vehicle_maintenance (vehicle_id, date, kind, description, cost_cents, odometer_km, workshop, created_by)
		VALUES ($1, $2::date, $3, $4, $5, $6, $7, NULLIF($8, '')::uuid)
		RETURNING ` + columns
	queryUpdate = `
		UPDATE vehicle_maintenance
		SET date = $2::date, kind = $3, description = $4, cost_cents = $5, odometer_km = $6, workshop = $7
		WHERE id = $1
		RETURNING ` + columns
	queryDelete = `DELETE FROM vehicle_maintenance WHERE id = $1`
)

type scanner interface {
	Scan(dest ...any) error
}

type MaintenanceRepository struct {
	db *sql.DB
}

func NewMaintenanceRepository(db *sql.DB) *MaintenanceRepository {
	return &MaintenanceRepository{db: db}
}

func (r *MaintenanceRepository) List(ctx context.Context, vehicleID string) ([]maintenance.Record, error) {
	var ok bool
	if err := r.db.QueryRowContext(ctx, queryVehicle, vehicleID).Scan(&ok); err != nil {
		return nil, mapError(err, maintenance.ErrVehicleNotFound, "vehicle exists")
	}
	if !ok {
		return nil, maintenance.ErrVehicleNotFound
	}
	rows, err := r.db.QueryContext(ctx, queryList, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("list maintenance: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []maintenance.Record{}
	for rows.Next() {
		rec, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan maintenance: %w", err)
		}
		list = append(list, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list maintenance: %w", err)
	}
	return list, nil
}

func (r *MaintenanceRepository) Create(ctx context.Context, vehicleID, userID string, req maintenance.Request) (maintenance.Record, error) {
	rec, err := scan(r.db.QueryRowContext(ctx, queryCreate, vehicleID, req.Date, req.Kind, req.Description,
		req.CostCents, req.OdometerKm, req.Workshop, userID))
	return rec, mapError(err, maintenance.ErrVehicleNotFound, "create maintenance")
}

func (r *MaintenanceRepository) Update(ctx context.Context, id string, req maintenance.Request) (maintenance.Record, error) {
	rec, err := scan(r.db.QueryRowContext(ctx, queryUpdate, id, req.Date, req.Kind, req.Description,
		req.CostCents, req.OdometerKm, req.Workshop))
	return rec, mapError(err, maintenance.ErrNotFound, "update maintenance")
}

func (r *MaintenanceRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, queryDelete, id)
	if err != nil {
		return mapError(err, maintenance.ErrNotFound, "delete maintenance")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return maintenance.ErrNotFound
	}
	return nil
}

func scan(s scanner) (maintenance.Record, error) {
	var rec maintenance.Record
	var cost sql.NullInt64
	var km sql.NullInt32
	err := s.Scan(&rec.ID, &rec.VehicleID, &rec.Date, &rec.Kind, &rec.Description, &cost, &km, &rec.Workshop, &rec.CreatedAt)
	if err != nil {
		return rec, err
	}
	if cost.Valid {
		rec.CostCents = &cost.Int64
	}
	if km.Valid {
		v := int(km.Int32)
		rec.OdometerKm = &v
	}
	return rec, nil
}

// mapError: linha ausente ou id malformado viram notFound; veículo inexistente na criação também.
func mapError(err, notFound error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return notFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == invalidTextRepresentation || pgErr.Code == foreignKeyViolation) {
		return notFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
