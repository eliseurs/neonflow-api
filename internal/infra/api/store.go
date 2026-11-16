package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/pkg/middleware"
)

const (
	invalidTextRepresentation = "22P02"

	queryActiveRole = `SELECT role FROM users WHERE id = $1 AND active`
	queryAudit      = `INSERT INTO audit_log (user_id, role, method, route, path, status) VALUES ($1, $2, $3, $4, $5, $6)`
)

// store é o que os middlewares precisam do banco: o papel atual do usuário (RN-05.8) e a gravação da auditoria (RN-05.9).
type store struct {
	db *sql.DB
}

func (s store) ActiveRole(ctx context.Context, userID string) (string, error) {
	var role string
	err := s.db.QueryRowContext(ctx, queryActiveRole, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", middleware.ErrAccountNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return "", middleware.ErrAccountNotFound
	}
	if err != nil {
		return "", fmt.Errorf("active role: %w", err)
	}
	return role, nil
}

func (s store) Record(ctx context.Context, e middleware.AuditEntry) error {
	if _, err := s.db.ExecContext(ctx, queryAudit, e.UserID, e.Role, e.Method, e.Route, e.Path, e.Status); err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	return nil
}
