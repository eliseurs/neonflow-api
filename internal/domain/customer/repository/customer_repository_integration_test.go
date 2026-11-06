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
	"neonflow-api/internal/domain/customer"
	"neonflow-api/internal/domain/transfer"
	transferRepo "neonflow-api/internal/domain/transfer/repository"
	"neonflow-api/internal/pkg/timezone"
	"neonflow-api/internal/testutil"
)

func TestCustomerRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewCustomerRepository(database)
	bookings := bookingRepo.NewBookingRepository(database)
	transfers := transferRepo.NewTransferRepository(database)
	ctx := context.Background()

	ana, err := repo.Create(ctx, customer.Request{Name: "Ana Teste", WhatsApp: "5554999990001", Email: "ana@exemplo.test", Language: "pt"})
	require.NoError(t, err)
	assert.Equal(t, "Ana Teste", ana.Name)
	assert.Nil(t, ana.AnonymizedAt)
	bob, err := repo.Create(ctx, customer.Request{Name: "Bob Guest", Language: "en"})
	require.NoError(t, err)
	_, err = repo.Create(ctx, customer.Request{Name: "Repetido", WhatsApp: "5554999990001", Language: "pt"})
	assert.ErrorIs(t, err, customer.ErrWhatsAppTaken)
	_, err = repo.Create(ctx, customer.Request{Name: "Também sem WhatsApp", Language: "pt"})
	require.NoError(t, err, "sem WhatsApp não conflita")

	// Busca por nome, e-mail ou dígitos do WhatsApp.
	for _, q := range []struct{ text, digits string }{{"ana", ""}, {"exemplo.test", ""}, {"99990001", "99990001"}} {
		list, err := repo.List(ctx, q.text, q.digits)
		require.NoError(t, err)
		require.Len(t, list, 1, q.text)
		assert.Equal(t, ana.ID, list[0].ID)
	}
	all, err := repo.List(ctx, "", "")
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// Pedidos ligados ao cliente aparecem no histórico, mais recentes primeiro.
	var userID, vehicleID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Central', 'central@neonflow.test', 'h', 'central') RETURNING id`).Scan(&userID))
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents)
		VALUES ('TST1A23', 'Sedã Teste', 'economy', 'Neon Flow', 5, 'manual', 10000) RETURNING id`).Scan(&vehicleID))
	p, err := booking.ParsePeriod("2026-12-10T10:00", "2026-12-12T10:00")
	require.NoError(t, err)
	b, err := bookings.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Ana Teste", Period: p, CreatedBy: userID, CustomerID: ana.ID})
	require.NoError(t, err)
	assert.Equal(t, ana.ID, *b.CustomerID)
	tr, err := transfers.Create(ctx, transfer.NewTransfer{ClientName: "Ana Teste", PickupAt: time.Date(2026, 12, 20, 8, 0, 0, 0, timezone.Location),
		Origin: "Aeroporto POA", Destination: "Gramado", Passengers: 1, Notes: "chega com bebê", PriceCents: 45000, CustomerID: ana.ID, CreatedBy: userID})
	require.NoError(t, err)
	assert.Equal(t, ana.ID, *tr.CustomerID)

	missing := "00000000-0000-0000-0000-000000000000"
	other, err := booking.ParsePeriod("2027-01-10T10:00", "2027-01-11T10:00")
	require.NoError(t, err)
	_, err = bookings.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "X", Period: other, CreatedBy: userID, CustomerID: missing})
	assert.ErrorIs(t, err, booking.ErrCustomerNotFound)
	_, err = transfers.Create(ctx, transfer.NewTransfer{ClientName: "X", PickupAt: time.Now(), Origin: "A", Destination: "B", Passengers: 1,
		PriceCents: 1, CustomerID: missing, CreatedBy: userID})
	assert.ErrorIs(t, err, transfer.ErrCustomerNotFound)

	d, err := repo.Get(ctx, ana.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, d.BookingCount)
	assert.Equal(t, 1, d.TransferCount)
	require.Len(t, d.Orders, 2)
	assert.Equal(t, "transfer", d.Orders[0].Kind)
	assert.Equal(t, "Aeroporto POA → Gramado", d.Orders[0].Summary)
	assert.Equal(t, "reserva", d.Orders[1].Kind)
	assert.Equal(t, "Sedã Teste · 2 diárias", d.Orders[1].Summary)
	assert.Equal(t, int64(20000), d.Orders[1].TotalCents)

	updated, err := repo.Update(ctx, bob.ID, customer.Request{Name: "Bob Guest Jr", WhatsApp: "15550000000", Language: "en"})
	require.NoError(t, err)
	assert.Equal(t, "Bob Guest Jr", updated.Name)
	_, err = repo.Update(ctx, bob.ID, customer.Request{Name: "Bob", WhatsApp: "5554999990001", Language: "en"})
	assert.ErrorIs(t, err, customer.ErrWhatsAppTaken)

	// Anonimizar apaga nome e contatos, também nos pedidos; valores ficam.
	anon, err := repo.Anonymize(ctx, ana.ID)
	require.NoError(t, err)
	assert.Equal(t, customer.AnonymizedName, anon.Name)
	assert.Empty(t, anon.WhatsApp)
	assert.Empty(t, anon.Email)
	require.NotNil(t, anon.AnonymizedAt)
	bAfter, err := bookings.FindByID(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, customer.AnonymizedName, bAfter.ClientName)
	assert.Equal(t, int64(20000), bAfter.TotalCents)
	tAfter, err := transfers.FindByID(ctx, tr.ID)
	require.NoError(t, err)
	assert.Equal(t, customer.AnonymizedName, tAfter.ClientName)
	assert.Empty(t, tAfter.Notes)
	_, err = repo.Update(ctx, ana.ID, customer.Request{Name: "Volta", Language: "pt"})
	assert.ErrorIs(t, err, customer.ErrAnonymized)
	_, err = repo.Anonymize(ctx, ana.ID)
	assert.ErrorIs(t, err, customer.ErrAnonymized)
	last, err := repo.List(ctx, "", "")
	require.NoError(t, err)
	assert.Equal(t, ana.ID, last[len(last)-1].ID, "anonimizados por último")

	for _, id := range []string{missing, "nao-e-uuid"} {
		_, err = repo.Get(ctx, id)
		assert.ErrorIs(t, err, customer.ErrNotFound)
		_, err = repo.Update(ctx, id, customer.Request{Name: "X", Language: "pt"})
		assert.ErrorIs(t, err, customer.ErrNotFound)
		_, err = repo.Anonymize(ctx, id)
		assert.ErrorIs(t, err, customer.ErrNotFound)
	}

	require.NoError(t, database.Close())
	_, err = repo.List(ctx, "", "")
	assert.Error(t, err)
	_, err = repo.Create(ctx, customer.Request{Name: "X", Language: "pt"})
	assert.Error(t, err)
}
