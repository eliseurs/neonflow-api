package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/photo"
)

const (
	invalidTextRepresentation = "22P02"

	photoColumns = `id, owner_type, owner_id, stage, storage_key, content_type, size_bytes, position, created_at`

	queryBookingStage = `SELECT stage FROM bookings WHERE id = $1`
	queryVehicle      = `SELECT EXISTS (SELECT 1 FROM vehicles WHERE id = $1)`
	queryCreate       = `
		INSERT INTO photos (owner_type, owner_id, stage, storage_key, content_type, size_bytes, position, created_by)
		VALUES ($1, $2, $3, $4, $5, $6,
		        (SELECT coalesce(max(position) + 1, 0) FROM photos WHERE owner_type = $1 AND owner_id = $2 AND stage = $3),
		        NULLIF($7, '')::uuid)
		RETURNING ` + photoColumns
	queryList = `
		SELECT ` + photoColumns + ` FROM photos
		WHERE owner_type = $1 AND owner_id = $2 AND ($3 = '' OR stage = $3)
		ORDER BY stage, position, created_at`
	queryDelete       = `DELETE FROM photos WHERE id = $1 RETURNING ` + photoColumns
	queryFind         = `SELECT ` + photoColumns + ` FROM photos WHERE id = $1`
	queryVehiclePhoto = `SELECT ` + photoColumns + ` FROM photos WHERE id = $1 AND owner_type = 'veiculo'`
	// Site público: carros fora de manutenção, Economy antes de Premium, com a primeira foto (capa).
	publicCar = `
		SELECT v.id, v.name, v.category, v.seats, v.gear, v.year, v.color, v.fuel, v.doors,
		       		       array_to_json(v.features), v.description, coalesce(v.min_days, 1),
		       coalesce((SELECT json_agg(p.id ORDER BY p.position, p.created_at) FROM photos p
		                 WHERE p.owner_type = 'veiculo' AND p.owner_id = v.id), '[]')
		FROM vehicles v
		WHERE NOT v.in_maintenance`
	queryPublicFleet = publicCar + ` ORDER BY v.category = 'premium', v.name`
	queryPublicCar   = publicCar + ` AND v.id = $1`
	// Capa: a escolhida vai para a posição 0 e as demais descem uma.
	queryCover = `
		UPDATE photos o SET position = CASE WHEN o.id = $1 THEN 0 ELSE o.position + 1 END
		FROM photos p
		WHERE p.id = $1 AND p.owner_type = 'veiculo' AND o.owner_type = p.owner_type AND o.owner_id = p.owner_id`
)

type scanner interface {
	Scan(dest ...any) error
}

type PhotoRepository struct {
	db *sql.DB
}

func NewPhotoRepository(db *sql.DB) *PhotoRepository {
	return &PhotoRepository{db: db}
}

func (r *PhotoRepository) BookingStage(ctx context.Context, id string) (string, error) {
	var stage string
	err := r.db.QueryRowContext(ctx, queryBookingStage, id).Scan(&stage)
	if err != nil {
		return "", mapError(err, photo.ErrOwnerNotFound, "booking stage")
	}
	return stage, nil
}

func (r *PhotoRepository) VehicleExists(ctx context.Context, id string) (bool, error) {
	var ok bool
	if err := r.db.QueryRowContext(ctx, queryVehicle, id).Scan(&ok); err != nil {
		if mapped := mapError(err, photo.ErrOwnerNotFound, "vehicle exists"); errors.Is(mapped, photo.ErrOwnerNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("vehicle exists: %w", err)
	}
	return ok, nil
}

func (r *PhotoRepository) Create(ctx context.Context, n photo.NewPhoto) (photo.Photo, error) {
	p, err := scan(r.db.QueryRowContext(ctx, queryCreate, n.OwnerType, n.OwnerID, n.Stage, n.Key, n.ContentType, n.SizeBytes, n.CreatedBy))
	if err != nil {
		return p, fmt.Errorf("create photo: %w", err)
	}
	return p, nil
}

func (r *PhotoRepository) List(ctx context.Context, ownerType, ownerID, stage string) ([]photo.Photo, error) {
	rows, err := r.db.QueryContext(ctx, queryList, ownerType, ownerID, stage)
	if err != nil {
		return nil, fmt.Errorf("list photos: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []photo.Photo{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan photo: %w", err)
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list photos: %w", err)
	}
	return list, nil
}

func (r *PhotoRepository) Delete(ctx context.Context, id string) (photo.Photo, error) {
	p, err := scan(r.db.QueryRowContext(ctx, queryDelete, id))
	return p, mapError(err, photo.ErrNotFound, "delete photo")
}

func (r *PhotoRepository) SetCover(ctx context.Context, id string) (photo.Photo, error) {
	res, err := r.db.ExecContext(ctx, queryCover, id)
	if err != nil {
		return photo.Photo{}, mapError(err, photo.ErrNotFound, "cover photo")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return photo.Photo{}, photo.ErrNotFound
	}
	p, err := scan(r.db.QueryRowContext(ctx, queryFind, id))
	return p, mapError(err, photo.ErrNotFound, "find photo")
}

func (r *PhotoRepository) PublicFleet(ctx context.Context) ([]photo.Car, error) {
	rows, err := r.db.QueryContext(ctx, queryPublicFleet)
	if err != nil {
		return nil, fmt.Errorf("public fleet: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	cars := []photo.Car{}
	for rows.Next() {
		c, err := scanCar(rows)
		if err != nil {
			return nil, fmt.Errorf("scan car: %w", err)
		}
		c.PhotoIDs = nil // a lista só vai na página do carro
		cars = append(cars, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("public fleet: %w", err)
	}
	return cars, nil
}

func (r *PhotoRepository) PublicCar(ctx context.Context, id string) (photo.Car, error) {
	c, err := scanCar(r.db.QueryRowContext(ctx, queryPublicCar, id))
	return c, mapError(err, photo.ErrCarNotFound, "public car")
}

func scanCar(s scanner) (photo.Car, error) {
	var c photo.Car
	var year, doors sql.NullInt32
	var features, photos []byte
	err := s.Scan(&c.ID, &c.Name, &c.Category, &c.Seats, &c.Gear, &year, &c.Color, &c.Fuel, &doors,
		&features, &c.Description, &c.MinDays, &photos)
	if err != nil {
		return c, err
	}
	if year.Valid {
		v := int(year.Int32)
		c.Year = &v
	}
	if doors.Valid {
		v := int(doors.Int32)
		c.Doors = &v
	}
	c.Features, c.PhotoIDs = []string{}, []string{}
	if err := json.Unmarshal(features, &c.Features); err != nil {
		return c, fmt.Errorf("decode features: %w", err)
	}
	if err := json.Unmarshal(photos, &c.PhotoIDs); err != nil {
		return c, fmt.Errorf("decode photos: %w", err)
	}
	if len(c.PhotoIDs) > 0 {
		c.PhotoID = &c.PhotoIDs[0]
	}
	return c, nil
}

func (r *PhotoRepository) VehiclePhoto(ctx context.Context, id string) (photo.Photo, error) {
	p, err := scan(r.db.QueryRowContext(ctx, queryVehiclePhoto, id))
	return p, mapError(err, photo.ErrNotFound, "vehicle photo")
}

func scan(s scanner) (photo.Photo, error) {
	var p photo.Photo
	err := s.Scan(&p.ID, &p.OwnerType, &p.OwnerID, &p.Stage, &p.Key, &p.ContentType, &p.SizeBytes, &p.Position, &p.CreatedAt)
	return p, err
}

func mapError(err error, notFound error, op string) error {
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
