//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/fleet"
	"neonflow-api/internal/testutil"
)

func TestFleetRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewFleetRepository(database)
	ctx := context.Background()

	empty, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, empty)

	req := fleet.VehicleRequest{Plate: "TST1A23", Name: "Sedã Teste", Category: "economy", Owner: "Parceiro Teste",
		Seats: 5, Gear: "manual", DailyRateCents: 10000}
	created, err := repo.Create(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.False(t, created.InMaintenance)

	_, err = repo.Create(ctx, req)
	assert.ErrorIs(t, err, fleet.ErrPlateTaken)

	bad := req
	bad.Plate, bad.Category = "TST9Z99", "luxo"
	_, err = repo.Create(ctx, bad)
	assert.Error(t, err, "CHECK de categoria no banco")
	assert.NotErrorIs(t, err, fleet.ErrPlateTaken)

	req.InMaintenance = true
	req.DailyRateCents = 12000
	updated, err := repo.Update(ctx, created.ID, req)
	require.NoError(t, err)
	assert.True(t, updated.InMaintenance)
	assert.Equal(t, int64(12000), updated.DailyRateCents)
	assert.True(t, updated.UpdatedAt.After(created.UpdatedAt) || updated.UpdatedAt.Equal(created.UpdatedAt))

	got, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "TST1A23", got.Plate)

	list, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	_, err = repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, fleet.ErrVehicleNotFound)
	_, err = repo.FindByID(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, fleet.ErrVehicleNotFound)
	_, err = repo.Update(ctx, "00000000-0000-0000-0000-000000000000", req)
	assert.ErrorIs(t, err, fleet.ErrVehicleNotFound)

	require.NoError(t, database.Close())
	_, err = repo.List(ctx)
	assert.Error(t, err)
}
