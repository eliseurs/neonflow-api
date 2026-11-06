package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/user"
)

const (
	uniqueViolation           = "23505"
	invalidTextRepresentation = "22P02"

	userColumns = `id, name, email, role, active, created_at, updated_at`

	queryList     = `SELECT ` + userColumns + ` FROM users ORDER BY active DESC, name, email`
	queryFindByID = `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	queryCreate   = `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + userColumns
	// "active" ausente mantém o valor atual.
	queryUpdate = `
		UPDATE users
		SET name = $2, role = $3, active = coalesce($4, active), updated_at = now()
		WHERE id = $1
		RETURNING ` + userColumns
	queryPasswordHash = `SELECT password_hash FROM users WHERE id = $1`
	querySetPassword  = `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`
)

type scanner interface {
	Scan(dest ...any) error
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) List(ctx context.Context) ([]user.User, error) {
	rows, err := r.db.QueryContext(ctx, queryList)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	users := []user.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (user.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx, queryFindByID, id))
	return u, mapError(err, "find user")
}

func (r *UserRepository) Create(ctx context.Context, n authentication.NewUser) (user.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx, queryCreate, n.Name, n.Email, n.PasswordHash, n.Role))
	return u, mapError(err, "create user")
}

func (r *UserRepository) Update(ctx context.Context, id string, req user.UpdateRequest) (user.User, error) {
	var active sql.NullBool
	if req.Active != nil {
		active = sql.NullBool{Bool: *req.Active, Valid: true}
	}
	u, err := scanUser(r.db.QueryRowContext(ctx, queryUpdate, id, req.Name, req.Role, active))
	return u, mapError(err, "update user")
}

func (r *UserRepository) PasswordHash(ctx context.Context, id string) (string, error) {
	var hash string
	err := r.db.QueryRowContext(ctx, queryPasswordHash, id).Scan(&hash)
	return hash, mapError(err, "password hash")
}

func (r *UserRepository) SetPassword(ctx context.Context, id, hash string) error {
	res, err := r.db.ExecContext(ctx, querySetPassword, id, hash)
	if err != nil {
		return mapError(err, "set password")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return authentication.ErrUserNotFound
	}
	return nil
}

func scanUser(s scanner) (user.User, error) {
	var u user.User
	err := s.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Active, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

// mapError traduz linha inexistente e id mal formatado em ErrUserNotFound e e-mail repetido em ErrEmailTaken.
func mapError(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return authentication.ErrUserNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			return authentication.ErrEmailTaken
		case invalidTextRepresentation:
			return authentication.ErrUserNotFound
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
