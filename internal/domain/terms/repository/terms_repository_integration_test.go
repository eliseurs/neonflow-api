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
	"neonflow-api/internal/domain/terms"
	"neonflow-api/internal/testutil"
)

func TestTermsRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewTermsRepository(database)
	bookings := bookingRepo.NewBookingRepository(database)
	ctx := context.Background()

	var userID, vehicleID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Admin', 'admin@neonflow.test', 'h', 'admin') RETURNING id`).Scan(&userID))
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents)
		VALUES ('TST1A23', 'Sedã Teste', 'economy', 'Neon Flow', 5, 'manual', 10000) RETURNING id`).Scan(&vehicleID))
	p, err := booking.ParsePeriod("2026-12-10T10:00", "2026-12-12T10:00")
	require.NoError(t, err)
	b, err := bookings.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Cliente Teste", Period: p, CreatedBy: userID})
	require.NoError(t, err)

	_, err = repo.Current(ctx)
	assert.ErrorIs(t, err, terms.ErrNoTerms)
	v1, err := repo.CreateVersion(ctx, "Termos v1", userID)
	require.NoError(t, err)
	assert.Equal(t, 1, v1.Version)
	v2, err := repo.CreateVersion(ctx, "Termos v2", userID)
	require.NoError(t, err)
	assert.Equal(t, 2, v2.Version)
	list, err := repo.Versions(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, 2, list[0].Version, "mais nova primeiro")

	stage, err := repo.BookingStage(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, booking.StagePreparacao, stage)
	_, err = repo.BookingStage(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, terms.ErrBookingNotFound)

	token, hash, err := terms.NewToken()
	require.NoError(t, err)
	require.NoError(t, repo.CreateLink(ctx, b.ID, hash, time.Now().Add(time.Hour), userID))
	link, err := repo.LinkByHash(ctx, terms.HashToken(token))
	require.NoError(t, err)
	assert.Equal(t, b.ID, link.BookingID)
	_, err = repo.LinkByHash(ctx, terms.HashToken("outro"))
	assert.ErrorIs(t, err, terms.ErrLinkNotFound)
	_, oldHash, _ := terms.NewToken()
	require.NoError(t, repo.CreateLink(ctx, b.ID, oldHash, time.Now().Add(-time.Minute), userID))
	_, err = repo.LinkByHash(ctx, oldHash)
	assert.ErrorIs(t, err, terms.ErrLinkNotFound, "link vencido")

	pub, err := repo.Public(ctx, link)
	require.NoError(t, err)
	assert.Equal(t, "Cliente Teste", pub.ClientName)
	assert.Equal(t, "Sedã Teste", pub.VehicleName)
	assert.Equal(t, int64(20000), pub.TotalCents)
	assert.Equal(t, 2, pub.Terms.Version)
	assert.Nil(t, pub.Accepted)

	// Aceitar a versão antiga é recusado; a atual marca a conferência "terms" da reserva.
	_, err = repo.Accept(ctx, link, 1, "Cliente Teste")
	assert.ErrorIs(t, err, terms.ErrVersionChanged)
	acc, err := repo.Accept(ctx, link, 2, "Cliente Teste")
	require.NoError(t, err)
	assert.Equal(t, 2, acc.Version)
	after, err := bookings.FindByID(ctx, b.ID)
	require.NoError(t, err)
	assert.True(t, after.Checks[terms.TermsCheck])
	last, err := repo.LastAcceptance(ctx, b.ID)
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, "Cliente Teste", last.TypedName)
	pub, err = repo.Public(ctx, link)
	require.NoError(t, err)
	require.NotNil(t, pub.Accepted)

	none, err := repo.LastAcceptance(ctx, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	assert.Nil(t, none)
	_, err = repo.LastAcceptance(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, terms.ErrBookingNotFound)
	_, err = repo.Public(ctx, terms.LinkInfo{BookingID: "00000000-0000-0000-0000-000000000000"})
	assert.ErrorIs(t, err, terms.ErrLinkNotFound)

	require.NoError(t, database.Close())
	_, err = repo.Versions(ctx)
	assert.Error(t, err)
	_, err = repo.Current(ctx)
	assert.Error(t, err)
	_, err = repo.CreateVersion(ctx, "x", userID)
	assert.Error(t, err)
	assert.Error(t, repo.CreateLink(ctx, b.ID, "h", time.Now(), userID))
	_, err = repo.Accept(ctx, link, 2, "Cliente Teste")
	assert.Error(t, err)
	_, err = repo.LastAcceptance(ctx, b.ID)
	assert.Error(t, err)
}

func TestTermsRepository_AcceptWithoutTerms(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewTermsRepository(database)
	_, err := repo.Accept(context.Background(), terms.LinkInfo{}, 1, "Ana Teste")
	assert.ErrorIs(t, err, terms.ErrNoTerms)
}
