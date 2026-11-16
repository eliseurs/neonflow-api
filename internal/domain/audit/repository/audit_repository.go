package repository

import (
	"context"
	"database/sql"
	"fmt"

	"neonflow-api/internal/domain/audit"
	"neonflow-api/internal/pkg/timezone"
)

// Mais recentes primeiro. O filtro de usuário compara como texto para id mal formatado só não achar nada.
const queryList = `
	SELECT a.id, a.user_id, u.name, a.role, a.method, a.route, a.path, a.status, a.created_at
	FROM audit_log a JOIN users u ON u.id = a.user_id
	WHERE ($1 = '' OR a.user_id::text = $1)
	  AND (NOT $2 OR a.status = 403)
	ORDER BY a.created_at DESC
	LIMIT $3`

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) List(ctx context.Context, f audit.Filter) ([]audit.Entry, error) {
	rows, err := r.db.QueryContext(ctx, queryList, f.UserID, f.Denied, f.Limit)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []audit.Entry{}
	for rows.Next() {
		var e audit.Entry
		if err := rows.Scan(&e.ID, &e.UserID, &e.UserName, &e.Role, &e.Method, &e.Route, &e.Path, &e.Status, &e.At); err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		e.At = e.At.In(timezone.Location)
		list = append(list, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	return list, nil
}
