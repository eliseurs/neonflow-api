package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/checklist"
)

const (
	uniqueViolation           = "23505"
	invalidTextRepresentation = "22P02"

	itemColumns = `id, key, label, position, active, created_at, updated_at`

	queryList = `
		SELECT ` + itemColumns + `
		FROM checklist_items
		ORDER BY position, label
	`
	queryCreate = `
		INSERT INTO checklist_items (key, label, position, active)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + itemColumns
	// A chave não muda: reservas já conferidas continuam apontando para o mesmo item.
	queryUpdate = `
		UPDATE checklist_items
		SET label = $2, position = $3, active = coalesce($4, active), updated_at = now()
		WHERE id = $1
		RETURNING ` + itemColumns
)

type scanner interface {
	Scan(dest ...any) error
}

type ChecklistRepository struct {
	db *sql.DB
}

func NewChecklistRepository(db *sql.DB) *ChecklistRepository {
	return &ChecklistRepository{db: db}
}

func (r *ChecklistRepository) List(ctx context.Context) ([]checklist.Item, error) {
	rows, err := r.db.QueryContext(ctx, queryList)
	if err != nil {
		return nil, fmt.Errorf("list checklist items: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	items := []checklist.Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan checklist item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list checklist items: %w", err)
	}
	return items, nil
}

func (r *ChecklistRepository) Create(ctx context.Context, key string, req checklist.ItemRequest) (checklist.Item, error) {
	active := req.Active == nil || *req.Active
	it, err := scanItem(r.db.QueryRowContext(ctx, queryCreate, key, req.Label, req.Position, active))
	return it, mapError(err, "create checklist item")
}

func (r *ChecklistRepository) Update(ctx context.Context, id string, req checklist.ItemRequest) (checklist.Item, error) {
	var active sql.NullBool
	if req.Active != nil {
		active = sql.NullBool{Bool: *req.Active, Valid: true}
	}
	it, err := scanItem(r.db.QueryRowContext(ctx, queryUpdate, id, req.Label, req.Position, active))
	return it, mapError(err, "update checklist item")
}

func scanItem(s scanner) (checklist.Item, error) {
	var it checklist.Item
	err := s.Scan(&it.ID, &it.Key, &it.Label, &it.Position, &it.Active, &it.CreatedAt, &it.UpdatedAt)
	return it, err
}

// mapError traduz linha inexistente e id mal formatado em ErrItemNotFound e chave repetida em ErrKeyTaken.
func mapError(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return checklist.ErrItemNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			return checklist.ErrKeyTaken
		case invalidTextRepresentation:
			return checklist.ErrItemNotFound
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
