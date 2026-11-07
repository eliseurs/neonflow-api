package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/fleet"
)

type fakeRepo struct {
	saved    fleet.VehicleRequest
	id       string
	statsID  string
	since    time.Time
	stats    []fleet.Stats
	statsErr error
}

func (f *fakeRepo) Stats(_ context.Context, id string, since time.Time) ([]fleet.Stats, error) {
	f.statsID, f.since = id, since
	return f.stats, f.statsErr
}

func (f *fakeRepo) List(context.Context) ([]fleet.Vehicle, error) {
	return []fleet.Vehicle{{ID: "v1"}}, nil
}

func (f *fakeRepo) FindByID(_ context.Context, id string) (fleet.Vehicle, error) {
	if id != "v1" {
		return fleet.Vehicle{}, fleet.ErrVehicleNotFound
	}
	return fleet.Vehicle{ID: id}, nil
}

func (f *fakeRepo) Create(_ context.Context, v fleet.VehicleRequest) (fleet.Vehicle, error) {
	f.saved = v
	return fleet.Vehicle{ID: "new", Plate: v.Plate}, nil
}

func (f *fakeRepo) Update(_ context.Context, id string, v fleet.VehicleRequest) (fleet.Vehicle, error) {
	f.saved, f.id = v, id
	return fleet.Vehicle{ID: id, Plate: v.Plate}, nil
}

func request() fleet.VehicleRequest {
	return fleet.VehicleRequest{Plate: "tst1a23", Name: "Sedã", Category: "premium", Owner: "Parceiro",
		Seats: 5, Gear: "automatico", DailyRateCents: 1}
}

func TestFleetService(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{}
	svc := NewFleetService(repo)

	list, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	_, err = svc.Get(ctx, "v1")
	assert.NoError(t, err)
	_, err = svc.Get(ctx, "x")
	assert.ErrorIs(t, err, fleet.ErrVehicleNotFound)

	v, err := svc.Create(ctx, request())
	require.NoError(t, err)
	assert.Equal(t, "TST1A23", v.Plate, "repository recebe a placa normalizada")
	assert.Equal(t, "TST1A23", repo.saved.Plate)

	_, err = svc.Update(ctx, "v1", request())
	require.NoError(t, err)
	assert.Equal(t, "v1", repo.id)

	bad := request()
	bad.DailyRateCents = 0
	_, err = svc.Create(ctx, bad)
	assert.ErrorIs(t, err, fleet.ErrInvalidDailyRate)
	_, err = svc.Update(ctx, "v1", bad)
	assert.ErrorIs(t, err, fleet.ErrInvalidDailyRate)
}

func TestFleetService_Stats(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{stats: []fleet.Stats{{VehicleID: "v1"}, {VehicleID: "v2"}}}
	svc := NewFleetService(repo)
	svc.now = func() time.Time { return now }

	list, err := svc.Comparison(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "", repo.statsID, "toda a frota")
	assert.True(t, repo.since.Equal(now.Add(-365*24*time.Hour)), "últimos 12 meses")

	one, err := svc.VehicleStats(ctx, "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1", one.VehicleID)
	assert.Equal(t, "v1", repo.statsID)

	repo.stats = nil
	_, err = svc.VehicleStats(ctx, "x")
	assert.ErrorIs(t, err, fleet.ErrVehicleNotFound)
	repo.statsErr = errors.New("db")
	_, err = svc.VehicleStats(ctx, "v1")
	assert.Error(t, err)
}
