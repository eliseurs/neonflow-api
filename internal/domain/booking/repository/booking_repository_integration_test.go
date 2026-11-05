//go:build integration

package repository

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/testutil"
)

func seed(t *testing.T, db *sql.DB) (userID, vehicleID string) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Central', 'central@neonflow.test', 'h', 'central') RETURNING id`).Scan(&userID))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents)
		 VALUES ('TST1A23', 'Sedã Teste', 'economy', 'Parceiro Teste', 5, 'manual', 10000) RETURNING id`).Scan(&vehicleID))
	return userID, vehicleID
}

func period(t *testing.T, start, end string) booking.Period {
	t.Helper()
	p, err := booking.ParsePeriod(start, end)
	require.NoError(t, err)
	return p
}

func TestBookingRepository_CreateAndLifecycle(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)

	b, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Cliente Teste",
		Period: period(t, "2026-12-10", "2026-12-13"), Place: "Gramado", CreatedBy: userID})
	require.NoError(t, err)
	assert.Equal(t, booking.StagePreparacao, b.Stage)
	assert.Equal(t, "2026-12-10", b.StartDate)
	assert.Equal(t, "2026-12-13", b.EndDate)
	assert.Equal(t, 3, b.Days)
	assert.Equal(t, int64(30000), b.TotalCents)
	assert.Equal(t, userID, *b.CreatedBy)
	assert.WithinDuration(t, b.CreatedAt.Add(booking.PaymentWindow), b.ExpiresAt, 0)

	// RN-02.6: mudar a diária do veículo não altera a reserva.
	_, err = database.ExecContext(ctx, `UPDATE vehicles SET daily_rate_cents = 99900 WHERE id = $1`, vehicleID)
	require.NoError(t, err)
	got, err := repo.FindByID(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(30000), got.TotalCents)

	// RN-02.2: sobreposição recusada, datas adjacentes permitidas.
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Outro",
		Period: period(t, "2026-12-12", "2026-12-15"), CreatedBy: userID})
	assert.ErrorIs(t, err, booking.ErrPeriodUnavailable)
	adj, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Outro",
		Period: period(t, "2026-12-13", "2026-12-14"), CreatedBy: userID})
	require.NoError(t, err)

	avail, err := repo.Available(ctx, period(t, "2026-12-11", "2026-12-12"))
	require.NoError(t, err)
	assert.Empty(t, avail)
	avail, err = repo.Available(ctx, period(t, "2026-12-14", "2026-12-20"))
	require.NoError(t, err)
	assert.Len(t, avail, 1)

	withChecks, err := repo.SetChecks(ctx, b.ID, booking.StagePreparacao, map[string]bool{"cnh": true, "payment": true})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"cnh": true, "payment": true}, withChecks.Checks)
	_, err = repo.SetChecks(ctx, b.ID, booking.StageConfirmada, map[string]bool{"delivery": true})
	assert.ErrorIs(t, err, booking.ErrStageChanged)

	moved, err := repo.Transition(ctx, b.ID, booking.StagePreparacao, booking.StageConfirmada, userID)
	require.NoError(t, err)
	assert.Equal(t, booking.StageConfirmada, moved.Stage)
	_, err = repo.Transition(ctx, b.ID, booking.StagePreparacao, booking.StageConfirmada, userID)
	assert.ErrorIs(t, err, booking.ErrStageChanged, "segundo clique não avança de novo")

	// Cancelar libera o período (RN-02.5).
	_, err = repo.Transition(ctx, adj.ID, booking.StagePreparacao, booking.StageCancelada, userID)
	require.NoError(t, err)
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Terceiro",
		Period: period(t, "2026-12-13", "2026-12-14"), CreatedBy: userID})
	require.NoError(t, err)

	events, err := repo.Events(ctx, b.ID)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Nil(t, events[0].FromStage)
	assert.Equal(t, booking.StagePreparacao, events[0].ToStage)
	assert.Equal(t, booking.StagePreparacao, *events[1].FromStage)
	assert.Equal(t, userID, *events[1].UserID)

	all, err := repo.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, all, 3)
	byVehicle, err := repo.List(ctx, vehicleID)
	require.NoError(t, err)
	assert.Len(t, byVehicle, 3)
	none, err := repo.List(ctx, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestBookingRepository_Errors(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)
	p := period(t, "2026-12-10", "2026-12-11")

	_, err := repo.Create(ctx, booking.NewBooking{VehicleID: "00000000-0000-0000-0000-000000000000", ClientName: "X", Period: p, CreatedBy: userID})
	assert.ErrorIs(t, err, booking.ErrVehicleNotFound)
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: "nao-e-uuid", ClientName: "X", Period: p, CreatedBy: userID})
	assert.ErrorIs(t, err, booking.ErrVehicleNotFound)

	// RN-01.1: veículo em manutenção não aceita reserva e some da disponibilidade.
	_, err = database.ExecContext(ctx, `UPDATE vehicles SET in_maintenance = true WHERE id = $1`, vehicleID)
	require.NoError(t, err)
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "X", Period: p, CreatedBy: userID})
	assert.ErrorIs(t, err, booking.ErrVehicleInMaintenance)
	avail, err := repo.Available(ctx, p)
	require.NoError(t, err)
	assert.Empty(t, avail)

	_, err = repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, booking.ErrBookingNotFound)
	_, err = repo.FindByID(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, booking.ErrBookingNotFound)
	_, err = repo.Events(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, booking.ErrBookingNotFound)
	_, err = repo.Transition(ctx, "00000000-0000-0000-0000-000000000000", "preparacao", "confirmada", userID)
	assert.ErrorIs(t, err, booking.ErrStageChanged)

	require.NoError(t, database.Close())
	_, err = repo.List(ctx, "")
	assert.Error(t, err)
	_, err = repo.FindByID(ctx, "x")
	assert.Error(t, err)
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "X", Period: p})
	assert.Error(t, err)
	_, err = repo.SetChecks(ctx, "x", "preparacao", map[string]bool{"cnh": true})
	assert.Error(t, err)
	_, err = repo.Available(ctx, p)
	assert.Error(t, err)
}

// RN-02.11: pré-reserva vencida e não paga é cancelada e libera o período; paga não expira.
func TestBookingRepository_Expiration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)
	p := period(t, "2026-12-10", "2026-12-12")
	later := period(t, "2026-12-20", "2026-12-22")

	unpaid, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Sem pagamento", Period: p, CreatedBy: userID})
	require.NoError(t, err)
	paid, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Pago", Period: later, CreatedBy: userID})
	require.NoError(t, err)
	_, err = repo.SetChecks(ctx, paid.ID, booking.StagePreparacao, map[string]bool{"payment": true})
	require.NoError(t, err)

	_, err = database.ExecContext(ctx, `UPDATE bookings SET expires_at = now() - interval '1 minute'`)
	require.NoError(t, err)

	again, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Novo", Period: p, CreatedBy: userID})
	require.NoError(t, err, "o período da pré-reserva vencida fica livre")
	assert.Equal(t, booking.StagePreparacao, again.Stage)

	got, err := repo.FindByID(ctx, unpaid.ID)
	require.NoError(t, err)
	assert.Equal(t, booking.StageCancelada, got.Stage)
	events, err := repo.Events(ctx, unpaid.ID)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Nil(t, events[1].UserID, "expiração é automática")

	got, err = repo.FindByID(ctx, paid.ID)
	require.NoError(t, err)
	assert.Equal(t, booking.StagePreparacao, got.Stage)
}

// Cenário do plano: duas centrais reservam o mesmo período ao mesmo tempo; só uma consegue (PEN-24).
func TestBookingRepository_ConcurrentCreate(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)
	p := period(t, "2026-12-10", "2026-12-13")

	const attempts = 8
	var wg sync.WaitGroup
	errs := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Concorrente", Period: p, CreatedBy: userID})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	ok := 0
	for err := range errs {
		if err == nil {
			ok++
			continue
		}
		assert.ErrorIs(t, err, booking.ErrPeriodUnavailable)
	}
	assert.Equal(t, 1, ok)
}
