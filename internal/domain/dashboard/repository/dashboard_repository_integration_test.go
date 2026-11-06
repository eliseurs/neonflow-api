//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/booking"
	bookingRepo "neonflow-api/internal/domain/booking/repository"
	"neonflow-api/internal/domain/dashboard"
	"neonflow-api/internal/pkg/timezone"
	"neonflow-api/internal/testutil"
)

func TestDashboardRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewDashboardRepository(database)
	bookings := bookingRepo.NewBookingRepository(database)
	ctx := context.Background()

	var userID, v1, v2 string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Central', 'central@neonflow.test', 'h', 'central') RETURNING id`).Scan(&userID))
	for i, dest := range []*string{&v1, &v2} {
		require.NoError(t, database.QueryRowContext(ctx,
			`INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents)
			 VALUES ($1, 'Sedã Teste', 'economy', 'Neon Flow', 5, 'manual', 10000) RETURNING id`,
			[]string{"TST1A23", "TST2B34"}[i]).Scan(dest))
	}
	create := func(vehicle, start, end string) booking.Booking {
		t.Helper()
		p, err := booking.ParsePeriod(start, end)
		require.NoError(t, err)
		b, err := bookings.Create(ctx, booking.NewBooking{VehicleID: vehicle, ClientName: "Cliente Teste", Period: p, CreatedBy: userID})
		require.NoError(t, err)
		return b
	}

	// Pagamento marcado grava paid_at; desmarcado apaga (PEN-11).
	b1 := create(v1, "2026-12-10T10:00", "2026-12-12T10:00")
	paid, err := bookings.SetChecks(ctx, b1.ID, booking.StagePreparacao, map[string]bool{"payment": true})
	require.NoError(t, err)
	require.NotNil(t, paid.PaidAt)
	first := *paid.PaidAt
	again, err := bookings.SetChecks(ctx, b1.ID, booking.StagePreparacao, map[string]bool{"cnh": true})
	require.NoError(t, err)
	assert.True(t, first.Equal(*again.PaidAt), "outras conferências não mexem na data")
	unpaid, err := bookings.SetChecks(ctx, b1.ID, booking.StagePreparacao, map[string]bool{"payment": false})
	require.NoError(t, err)
	assert.Nil(t, unpaid.PaidAt)
	_, err = bookings.SetChecks(ctx, b1.ID, booking.StagePreparacao, map[string]bool{"payment": true})
	require.NoError(t, err)

	// b1 confirmada e em locação, com atraso registrado; b2 só pré-reserva (não entra no faturamento).
	_, err = bookings.Transition(ctx, b1.ID, booking.StagePreparacao, booking.StageConfirmada, userID, booking.Record{})
	require.NoError(t, err)
	_, err = bookings.Transition(ctx, b1.ID, booking.StageConfirmada, booking.StageEmLocacao, userID, booking.Record{})
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `UPDATE bookings SET extra_cents = 2500, excess_km_cents = 1500 WHERE id = $1`, b1.ID)
	require.NoError(t, err)
	create(v2, "2026-12-20T10:00", "2026-12-21T10:00")

	dec := time.Date(2026, 12, 1, 0, 0, 0, 0, timezone.Location)
	totals, err := repo.Totals(ctx, dec, dec.AddDate(0, 1, 0))
	require.NoError(t, err)
	assert.Equal(t, dashboard.Totals{PickupRevenueCents: 24000, PickupCount: 1, RentedToday: 1, FleetSize: 2}, dashboard.Totals{
		PickupRevenueCents: totals.PickupRevenueCents, PickupCount: totals.PickupCount, RentedToday: totals.RentedToday, FleetSize: totals.FleetSize,
	}, "retirada em dezembro: diárias + atraso + km; pré-reserva fora")

	// O pagamento foi marcado agora, então conta no mês corrente.
	now := time.Now().In(timezone.Location)
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, timezone.Location)
	current, err := repo.Totals(ctx, month, month.AddDate(0, 1, 0))
	require.NoError(t, err)
	assert.Equal(t, int64(20000), current.PaidCents)
	assert.Equal(t, 1, current.PaidCount)

	require.NoError(t, database.Close())
	_, err = repo.Totals(ctx, dec, dec.AddDate(0, 1, 0))
	assert.Error(t, err)
}
