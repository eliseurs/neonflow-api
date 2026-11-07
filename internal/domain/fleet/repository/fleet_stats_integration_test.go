//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/fleet"
	"neonflow-api/internal/testutil"
)

func TestFleetRepository_DetailsAndStats(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewFleetRepository(database)
	ctx := context.Background()
	year, doors := 2014, 4

	// Ficha: ano, cor, combustível, portas, itens e descrição.
	jetta, err := repo.Create(ctx, fleet.VehicleRequest{Plate: "TST1A23", Name: "Jetta Teste", Category: "economy", Owner: "Dono",
		Seats: 5, Gear: "automatico", DailyRateCents: 20000, Year: &year, Color: "Preto", Fuel: "flex", Doors: &doors,
		Features: []string{"ar_condicionado", "multimidia"}, Description: "Sedã confortável."})
	require.NoError(t, err)
	assert.Equal(t, 2014, *jetta.Year)
	assert.Equal(t, 4, *jetta.Doors)
	assert.Equal(t, "flex", jetta.Fuel)
	assert.Equal(t, []string{"ar_condicionado", "multimidia"}, jetta.Features)
	assert.Equal(t, "Sedã confortável.", jetta.Description)
	mobi, err := repo.Create(ctx, fleet.VehicleRequest{Plate: "TST2B34", Name: "Mobi Teste", Category: "economy", Owner: "Dono",
		Seats: 5, Gear: "manual", DailyRateCents: 10000})
	require.NoError(t, err)
	assert.Nil(t, mobi.Year)
	assert.Equal(t, []string{}, mobi.Features)

	// Duas locações do Jetta nos últimos dias (uma devolvida com km), uma cancelada e uma futura que não contam.
	var userID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Central', 'central@neonflow.test', 'h', 'central') RETURNING id`).Scan(&userID))
	insert := func(stage string, start time.Time, days int, total int64, deliveredKm, returnedKm any) {
		t.Helper()
		_, err := database.ExecContext(ctx, `INSERT INTO bookings (vehicle_id, client_name, start_at, end_at, place, stage, days,
			daily_rate_cents, total_cents, expires_at, created_by, delivered_km, returned_km, delivered_at, returned_at)
			VALUES ($1, 'Cliente', $2, $3, '', $4, $5, 10000, $6, now(), $7, $8, $9, $2, $3)`,
			jetta.ID, start, start.Add(time.Duration(days)*24*time.Hour), stage, days, total, userID, deliveredKm, returnedKm)
		require.NoError(t, err)
	}
	now := time.Now()
	insert("encerrada", now.Add(-10*24*time.Hour), 2, 40000, 1000, 1350)
	insert("encerrada", now.Add(-5*24*time.Hour), 1, 20000, nil, nil)
	insert("cancelada", now.Add(-3*24*time.Hour), 1, 20000, nil, nil)
	insert("confirmada", now.Add(5*24*time.Hour), 1, 20000, nil, nil)
	_, err = database.ExecContext(ctx, `INSERT INTO vehicle_maintenance (vehicle_id, date, kind, description, cost_cents)
		VALUES ($1, current_date - 2, 'preventiva', 'Troca de óleo', 30000),
		       ($1, current_date - 400, 'corretiva', 'Fora do período', 99999)`, jetta.ID)
	require.NoError(t, err)

	since := now.Add(-fleet.StatsWindow)
	all, err := repo.Stats(ctx, "", since)
	require.NoError(t, err)
	require.Len(t, all, 2)
	s := all[0]
	assert.Equal(t, jetta.ID, s.VehicleID)
	assert.Equal(t, 2, s.Bookings, "cancelada e futura não contam")
	assert.Equal(t, 3, s.RentedDays)
	assert.Equal(t, int64(60000), s.RevenueCents)
	assert.Equal(t, 350, s.KmDriven)
	assert.Equal(t, 1, s.MaintenanceCount, "manutenção fora do período não conta")
	assert.Equal(t, int64(30000), s.MaintenanceCents)
	assert.Equal(t, int64(30000), s.ResultCents)
	require.NotNil(t, s.LastMaintenance)
	assert.InDelta(t, 3.0/365*100, s.OccupancyPercent, 0.15, "3 dias em 365")
	assert.Zero(t, all[1].Bookings)
	assert.Nil(t, all[1].LastMaintenance)

	one, err := repo.Stats(ctx, mobi.ID, since)
	require.NoError(t, err)
	require.Len(t, one, 1)
	assert.Equal(t, mobi.ID, one[0].VehicleID)
	none, err := repo.Stats(ctx, "nao-e-uuid", since)
	require.NoError(t, err)
	assert.Empty(t, none)

	require.NoError(t, database.Close())
	_, err = repo.Stats(ctx, "", since)
	assert.Error(t, err)
}
