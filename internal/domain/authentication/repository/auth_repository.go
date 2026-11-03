package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/authentication"
)

const (
	uniqueViolation = "23505"

	queryFindByEmail = `
		SELECT id, name, email, role, password_hash, created_at, updated_at
		FROM users
		WHERE lower(email) = $1
	`
	queryFindByID = `
		SELECT id, name, email, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	queryCreate = `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, email, role, created_at, updated_at
	`
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) FindByEmail(ctx context.Context, email string) (authentication.Credentials, error) {
	var c authentication.Credentials
	err := r.db.QueryRowContext(ctx, queryFindByEmail, authentication.NormalizeEmail(email)).
		Scan(&c.ID, &c.Name, &c.Email, &c.Role, &c.PasswordHash, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c, authentication.ErrUserNotFound
	}
	if err != nil {
		return c, fmt.Errorf("find user by email: %w", err)
	}
	return c, nil
}

func (r *AuthRepository) FindByID(ctx context.Context, id string) (authentication.User, error) {
	var u authentication.User
	err := r.db.QueryRowContext(ctx, queryFindByID, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return u, authentication.ErrUserNotFound
	}
	if err != nil {
		return u, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}

func (r *AuthRepository) Create(ctx context.Context, n authentication.NewUser) (authentication.User, error) {
	var u authentication.User
	err := r.db.QueryRowContext(ctx, queryCreate, n.Name, n.Email, n.PasswordHash, n.Role).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return u, authentication.ErrEmailTaken
	}
	if err != nil {
		return u, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}
