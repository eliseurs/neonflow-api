//go:build integration

package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/pkg/middleware"
	"neonflow-api/internal/testutil"
)

func TestStore_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../sql/migrations")
	s := store{db: database}
	ctx := context.Background()

	var id string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Central Teste', 'central@neonflow.test', 'h', 'central') RETURNING id`).Scan(&id))

	role, err := s.ActiveRole(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "central", role)

	_, err = s.ActiveRole(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, middleware.ErrAccountNotFound)
	_, err = s.ActiveRole(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, middleware.ErrAccountNotFound)

	entry := middleware.AuditEntry{UserID: id, Role: role, Method: "POST", Route: "/api/bookings/:id/cancel", Path: "/api/bookings/b1/cancel", Status: 200}
	require.NoError(t, s.Record(ctx, entry))
	var n int
	require.NoError(t, database.QueryRowContext(ctx, `SELECT count(*) FROM audit_log WHERE user_id = $1 AND status = 200`, id).Scan(&n))
	assert.Equal(t, 1, n)

	// A trilha é só de inclusão (RN-05.9).
	_, err = database.ExecContext(ctx, `UPDATE audit_log SET status = 500`)
	assert.Error(t, err)
	_, err = database.ExecContext(ctx, `DELETE FROM audit_log`)
	assert.Error(t, err)

	_, err = database.ExecContext(ctx, `UPDATE users SET active = false WHERE id = $1`, id)
	require.NoError(t, err)
	_, err = s.ActiveRole(ctx, id)
	assert.ErrorIs(t, err, middleware.ErrAccountNotFound, "desativado perde o acesso na hora (RN-05.8)")

	entry.UserID = "00000000-0000-0000-0000-000000000000"
	assert.Error(t, s.Record(ctx, entry), "usuário inexistente")

	require.NoError(t, database.Close())
	_, err = s.ActiveRole(ctx, id)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, middleware.ErrAccountNotFound)
}
