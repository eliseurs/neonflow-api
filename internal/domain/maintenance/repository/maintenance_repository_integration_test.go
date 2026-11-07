//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/maintenance"
	"neonflow-api/internal/testutil"
)

func TestMaintenanceRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewMaintenanceRepository(database)
	ctx := context.Background()
	var userID, vehicleID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Central', 'central@neonflow.test', 'h', 'central') RETURNING id`).Scan(&userID))
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents)
		VALUES ('TST1A23', 'Sedã Teste', 'economy', 'Dono', 5, 'manual', 10000) RETURNING id`).Scan(&vehicleID))
	cost, km := int64(30000), 61000

	empty, err := repo.List(ctx, vehicleID)
	require.NoError(t, err)
	assert.Empty(t, empty)

	old, err := repo.Create(ctx, vehicleID, userID, maintenance.Request{Date: "2026-05-10", Kind: "pneus", Description: "4 pneus"})
	require.NoError(t, err)
	assert.Nil(t, old.CostCents)
	recent, err := repo.Create(ctx, vehicleID, userID, maintenance.Request{Date: "2026-10-01", Kind: "preventiva",
		Description: "Troca de óleo", CostCents: &cost, OdometerKm: &km, Workshop: "Oficina Teste"})
	require.NoError(t, err)
	assert.Equal(t, "2026-10-01", recent.Date)
	assert.Equal(t, int64(30000), *recent.CostCents)
	assert.Equal(t, 61000, *recent.OdometerKm)

	list, err := repo.List(ctx, vehicleID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, recent.ID, list[0].ID, "mais recente primeiro")

	upd, err := repo.Update(ctx, old.ID, maintenance.Request{Date: "2026-05-11", Kind: "pneus", Description: "4 pneus novos", CostCents: &cost})
	require.NoError(t, err)
	assert.Equal(t, "2026-05-11", upd.Date)
	assert.Equal(t, "4 pneus novos", upd.Description)

	require.NoError(t, repo.Delete(ctx, old.ID))
	assert.ErrorIs(t, repo.Delete(ctx, old.ID), maintenance.ErrNotFound)
	missing := "00000000-0000-0000-0000-000000000000"
	for _, id := range []string{missing, "nao-e-uuid"} {
		_, err = repo.List(ctx, id)
		assert.ErrorIs(t, err, maintenance.ErrVehicleNotFound)
		_, err = repo.Create(ctx, id, userID, maintenance.Request{Date: "2026-10-01", Kind: "outros", Description: "x"})
		assert.ErrorIs(t, err, maintenance.ErrVehicleNotFound)
		_, err = repo.Update(ctx, id, maintenance.Request{Date: "2026-10-01", Kind: "outros", Description: "x"})
		assert.ErrorIs(t, err, maintenance.ErrNotFound)
		assert.ErrorIs(t, repo.Delete(ctx, id), maintenance.ErrNotFound)
	}

	require.NoError(t, database.Close())
	_, err = repo.List(ctx, vehicleID)
	assert.Error(t, err)
	_, err = repo.Create(ctx, vehicleID, userID, maintenance.Request{Date: "2026-10-01", Kind: "outros", Description: "x"})
	assert.Error(t, err)
	assert.Error(t, repo.Delete(ctx, recent.ID))
}
