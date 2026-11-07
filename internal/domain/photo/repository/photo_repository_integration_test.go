//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/booking"
	bookingRepo "neonflow-api/internal/domain/booking/repository"
	"neonflow-api/internal/domain/photo"
	"neonflow-api/internal/testutil"
)

func TestPhotoRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewPhotoRepository(database)
	bookings := bookingRepo.NewBookingRepository(database)
	ctx := context.Background()

	var userID, vehicleID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Central', 'central@neonflow.test', 'h', 'central') RETURNING id`).Scan(&userID))
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents)
		VALUES ('TST1A23', 'Sedã Teste', 'economy', 'Neon Flow', 5, 'manual', 10000) RETURNING id`).Scan(&vehicleID))
	p, err := booking.ParsePeriod("2026-12-10T10:00", "2026-12-12T10:00")
	require.NoError(t, err)
	b, err := bookings.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Cliente", Period: p, CreatedBy: userID})
	require.NoError(t, err)

	ok, err := repo.VehicleExists(ctx, vehicleID)
	require.NoError(t, err)
	assert.True(t, ok)
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		ok, err = repo.VehicleExists(ctx, id)
		require.NoError(t, err)
		assert.False(t, ok)
		_, err = repo.BookingStage(ctx, id)
		assert.ErrorIs(t, err, photo.ErrOwnerNotFound)
	}
	stage, err := repo.BookingStage(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, booking.StagePreparacao, stage)

	// Posições crescem por dono; capa vai para a frente.
	add := func(owner, id, st, key string) photo.Photo {
		t.Helper()
		out, err := repo.Create(ctx, photo.NewPhoto{OwnerType: owner, OwnerID: id, Stage: st, Key: key, ContentType: "image/jpeg", SizeBytes: 10, CreatedBy: userID})
		require.NoError(t, err)
		return out
	}
	a := add(photo.OwnerVehicle, vehicleID, "", "veiculos/v/a.jpg")
	bb := add(photo.OwnerVehicle, vehicleID, "", "veiculos/v/b.jpg")
	assert.Equal(t, 0, a.Position)
	assert.Equal(t, 1, bb.Position)
	cover, err := repo.SetCover(ctx, bb.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, cover.Position)
	list, err := repo.List(ctx, photo.OwnerVehicle, vehicleID, "")
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, bb.ID, list[0].ID, "capa primeiro")
	assert.Equal(t, "veiculos/v/b.jpg", list[0].Key)

	// Fotos da devolução contam para o item do checklist.
	n, err := bookings.ReturnPhotos(ctx, b.ID)
	require.NoError(t, err)
	assert.Zero(t, n)
	add(photo.OwnerBooking, b.ID, photo.StageDelivery, "reservas/b/entrega/a.jpg")
	add(photo.OwnerBooking, b.ID, photo.StageReturn, "reservas/b/devolucao/a.jpg")
	n, err = bookings.ReturnPhotos(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	ret, err := repo.List(ctx, photo.OwnerBooking, b.ID, photo.StageReturn)
	require.NoError(t, err)
	assert.Len(t, ret, 1)
	all, err := repo.List(ctx, photo.OwnerBooking, b.ID, "")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// Reserva não tem capa; chave repetida e etapa errada são recusadas pelo banco.
	_, err = repo.SetCover(ctx, ret[0].ID)
	assert.ErrorIs(t, err, photo.ErrNotFound)
	_, err = repo.Create(ctx, photo.NewPhoto{OwnerType: photo.OwnerVehicle, OwnerID: vehicleID, Key: "veiculos/v/a.jpg", ContentType: "image/jpeg", SizeBytes: 1})
	assert.Error(t, err)
	_, err = repo.Create(ctx, photo.NewPhoto{OwnerType: photo.OwnerVehicle, OwnerID: vehicleID, Stage: photo.StageReturn, Key: "x.jpg", ContentType: "image/jpeg", SizeBytes: 1})
	assert.Error(t, err)

	// Site público: só veículo fora de manutenção, com a capa; foto de reserva não é pública.
	var maintID string
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents, in_maintenance)
		VALUES ('TST9Z99', 'Na Oficina', 'premium', 'Neon Flow', 5, 'automatico', 10000, true) RETURNING id`).Scan(&maintID))
	var premiumID string
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents)
		VALUES ('TST8Y88', 'Alfa Premium', 'premium', 'Neon Flow', 5, 'automatico', 30000) RETURNING id`).Scan(&premiumID))
	fleet, err := repo.PublicFleet(ctx)
	require.NoError(t, err)
	require.Len(t, fleet, 2)
	assert.Equal(t, vehicleID, fleet[0].ID, "Economy antes de Premium")
	assert.Equal(t, bb.ID, *fleet[0].PhotoID, "capa")
	assert.Equal(t, premiumID, fleet[1].ID)
	assert.Nil(t, fleet[1].PhotoID)
	assert.Nil(t, fleet[0].PhotoIDs, "a lista de fotos só vai na página do carro")
	assert.Equal(t, []string{}, fleet[1].Features)
	assert.Equal(t, 1, fleet[0].MinDays, "sem mínimo configurado vale 1")

	// Página do carro: todas as fotos, a capa primeiro; em manutenção não aparece.
	car, err := repo.PublicCar(ctx, vehicleID)
	require.NoError(t, err)
	assert.Equal(t, []string{bb.ID, a.ID}, car.PhotoIDs)
	assert.Equal(t, bb.ID, *car.PhotoID)
	for _, id := range []string{maintID, "00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		_, err = repo.PublicCar(ctx, id)
		assert.ErrorIs(t, err, photo.ErrCarNotFound, id)
	}
	vp, err := repo.VehiclePhoto(ctx, bb.ID)
	require.NoError(t, err)
	assert.Equal(t, "veiculos/v/b.jpg", vp.Key)
	_, err = repo.VehiclePhoto(ctx, ret[0].ID)
	assert.ErrorIs(t, err, photo.ErrNotFound, "foto de reserva não é pública")
	_, err = repo.VehiclePhoto(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, photo.ErrNotFound)

	deleted, err := repo.Delete(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "veiculos/v/a.jpg", deleted.Key)
	for _, id := range []string{a.ID, "nao-e-uuid"} {
		_, err = repo.Delete(ctx, id)
		assert.ErrorIs(t, err, photo.ErrNotFound)
		_, err = repo.SetCover(ctx, id)
		assert.ErrorIs(t, err, photo.ErrNotFound)
	}

	require.NoError(t, database.Close())
	_, err = repo.List(ctx, photo.OwnerVehicle, vehicleID, "")
	assert.Error(t, err)
	_, err = repo.BookingStage(ctx, b.ID)
	assert.Error(t, err)
	_, err = repo.VehicleExists(ctx, vehicleID)
	assert.Error(t, err)
	_, err = repo.PublicFleet(ctx)
	assert.Error(t, err)
	_, err = bookings.ReturnPhotos(ctx, b.ID)
	assert.Error(t, err)
}
