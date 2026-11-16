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

	// Parceiro desativado não entra nem mantém a sessão, como o usuário desativado (RN-05.8).
	accountFilter = `
		FROM users u
		LEFT JOIN partners p ON p.id = u.partner_id
		WHERE u.active AND (u.partner_id IS NULL OR p.active)
	`
	userColumns = `u.id, u.name, u.email, u.role, u.partner_id, u.customer_id, u.created_at, u.updated_at`

	queryFindByEmail = `SELECT ` + userColumns + `, u.password_hash, u.email_verified_at IS NOT NULL` + accountFilter + ` AND lower(u.email) = $1`
	queryFindByID    = `SELECT ` + userColumns + accountFilter + ` AND u.id = $1`
	queryCreate      = `
		INSERT INTO users AS u (name, email, password_hash, role, email_verified_at)
		VALUES ($1, $2, $3, $4, now())
		RETURNING ` + userColumns
)

type scanner interface {
	Scan(dest ...any) error
}

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) FindByEmail(ctx context.Context, email string) (authentication.Credentials, error) {
	var c authentication.Credentials
	var partnerID, customerID sql.NullString
	err := r.db.QueryRowContext(ctx, queryFindByEmail, authentication.NormalizeEmail(email)).
		Scan(&c.ID, &c.Name, &c.Email, &c.Role, &partnerID, &customerID, &c.CreatedAt, &c.UpdatedAt, &c.PasswordHash, &c.EmailVerified)
	c.PartnerID, c.CustomerID = nullable(partnerID), nullable(customerID)
	if errors.Is(err, sql.ErrNoRows) {
		return c, authentication.ErrUserNotFound
	}
	if err != nil {
		return c, fmt.Errorf("find user by email: %w", err)
	}
	return c, nil
}

func (r *AuthRepository) FindByID(ctx context.Context, id string) (authentication.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx, queryFindByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return u, authentication.ErrUserNotFound
	}
	if err != nil {
		return u, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}

func (r *AuthRepository) Create(ctx context.Context, n authentication.NewUser) (authentication.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx, queryCreate, n.Name, n.Email, n.PasswordHash, n.Role))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return u, authentication.ErrEmailTaken
	}
	if err != nil {
		return u, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func scanUser(s scanner) (authentication.User, error) {
	var u authentication.User
	var partnerID, customerID sql.NullString
	err := s.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &partnerID, &customerID, &u.CreatedAt, &u.UpdatedAt)
	u.PartnerID, u.CustomerID = nullable(partnerID), nullable(customerID)
	return u, err
}

func nullable(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}
