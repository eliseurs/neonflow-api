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

func TestFleetRepository_Status(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewFleetRepository(database)
	ctx := context.Background()

	vehicle := func(plate string, maintenance bool) string {
		v, err := repo.Create(ctx, fleet.VehicleRequest{Plate: plate, Name: "Carro " + plate, Category: "economy",
			Owner: "Parceiro Teste", Seats: 5, Gear: "manual", DailyRateCents: 10000, InMaintenance: maintenance})
		require.NoError(t, err)
		return v.ID
	}
	// Retirada e devolução em dias relativos a agora, como a consulta.
	book := func(vehicleID, stage string, startOffset, endOffset int, expiresIn string, paid bool) string {
		var id string
		require.NoError(t, database.QueryRowContext(ctx, `
			INSERT INTO bookings (vehicle_id, client_name, start_at, end_at, stage, days,
			                      daily_rate_cents, total_cents, checks, expires_at)
			VALUES ($1, 'Cliente Teste', now() + make_interval(days => $2::int),
			        now() + make_interval(days => $3::int), $4, greatest($3::int - $2::int, 1),
			        10000, 10000, jsonb_build_object('payment', $5::boolean), now() + $6::interval)
			RETURNING id`, vehicleID, startOffset, endOffset, stage, paid, expiresIn).Scan(&id))
		return id
	}

	livre := vehicle("TST1A01", false)
	book(livre, "confirmada", -5, -2, "1 day", true) // já terminou
	book(livre, "encerrada", 3, 5, "1 day", true)    // encerrada não conta

	reservado := vehicle("TST1A02", false)
	later := book(reservado, "confirmada", 10, 12, "1 day", true)
	next := book(reservado, "confirmada", 2, 4, "1 day", true)

	emLocacao := vehicle("TST1A03", false)
	book(emLocacao, "em_locacao", -1, 2, "1 day", true)

	aguardando := vehicle("TST1A04", false)
	book(aguardando, "devolvida", -3, 0, "1 day", true)

	manutencao := vehicle("TST1A05", true)
	book(manutencao, "em_locacao", -1, 2, "1 day", true)

	vencida := vehicle("TST1A06", false)
	book(vencida, "preparacao", 1, 3, "-1 hour", false) // pré-reserva vencida sem pagamento (RN-02.11)

	paga := vehicle("TST1A07", false)
	pagaID := book(paga, "preparacao", 1, 3, "-1 hour", true) // venceu o prazo, mas foi paga

	list, err := repo.List(ctx)
	require.NoError(t, err)
	byID := map[string]fleet.Vehicle{}
	for _, v := range list {
		byID[v.ID] = v
	}

	assert.Equal(t, fleet.StatusLivre, byID[livre].Status)
	assert.Nil(t, byID[livre].NextBooking)
	assert.Equal(t, fleet.StatusReservado, byID[reservado].Status)
	require.NotNil(t, byID[reservado].NextBooking)
	assert.Equal(t, next, byID[reservado].NextBooking.ID, "a próxima é a de início mais cedo")
	assert.NotEqual(t, later, byID[reservado].NextBooking.ID)
	assert.Equal(t, "confirmada", byID[reservado].NextBooking.Stage)
	assert.Equal(t, "Cliente Teste", byID[reservado].NextBooking.ClientName)
	assert.Len(t, byID[reservado].NextBooking.StartDate, len("2006-01-02"))
	assert.Equal(t, "America/Sao_Paulo", byID[reservado].NextBooking.StartAt.Location().String())
	assert.True(t, byID[reservado].NextBooking.EndAt.After(byID[reservado].NextBooking.StartAt))
	assert.Equal(t, fleet.StatusEmLocacao, byID[emLocacao].Status)
	assert.Equal(t, fleet.StatusAguardandoChecklist, byID[aguardando].Status)
	assert.Equal(t, fleet.StatusManutencao, byID[manutencao].Status, "manutenção prevalece (RN-01.1)")
	assert.Equal(t, fleet.StatusLivre, byID[vencida].Status)
	assert.Equal(t, fleet.StatusReservado, byID[paga].Status)
	assert.Equal(t, pagaID, byID[paga].NextBooking.ID)

	one, err := repo.FindByID(ctx, aguardando)
	require.NoError(t, err)
	assert.Equal(t, fleet.StatusAguardandoChecklist, one.Status)
}
