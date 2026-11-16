//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/audit"
	"neonflow-api/internal/testutil"
)

func TestAuditRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewAuditRepository(database)
	ctx := context.Background()

	var admin, central string
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO users (name, email, password_hash, role) VALUES ('Admin', 'admin@neonflow.test', 'h', 'admin') RETURNING id`).Scan(&admin))
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO users (name, email, password_hash, role) VALUES ('Central', 'central@neonflow.test', 'h', 'central') RETURNING id`).Scan(&central))
	_, err := database.ExecContext(ctx, `
		INSERT INTO audit_log (user_id, role, method, route, path, status, created_at) VALUES
		($1, 'admin', 'POST', '/api/users', '/api/users', 201, now() - interval '2 minutes'),
		($2, 'central', 'GET', '/api/users', '/api/users', 403, now() - interval '1 minute'),
		($2, 'central', 'POST', '/api/bookings', '/api/bookings', 201, now())`, admin, central)
	require.NoError(t, err)

	all, err := repo.List(ctx, audit.Filter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, all, 3)
	assert.Equal(t, "/api/bookings", all[0].Route, "mais recente primeiro")
	assert.Equal(t, "Central", all[0].UserName)

	mine, err := repo.List(ctx, audit.Filter{UserID: admin, Limit: 10})
	require.NoError(t, err)
	require.Len(t, mine, 1)
	assert.Equal(t, 201, mine[0].Status)

	denied, err := repo.List(ctx, audit.Filter{Denied: true, Limit: 10})
	require.NoError(t, err)
	require.Len(t, denied, 1)
	assert.Equal(t, central, denied[0].UserID)

	limited, err := repo.List(ctx, audit.Filter{Limit: 1})
	require.NoError(t, err)
	assert.Len(t, limited, 1)

	none, err := repo.List(ctx, audit.Filter{UserID: "nao-e-uuid", Limit: 10})
	require.NoError(t, err)
	assert.Empty(t, none)

	require.NoError(t, database.Close())
	_, err = repo.List(ctx, audit.Filter{Limit: 10})
	assert.Error(t, err)
}
