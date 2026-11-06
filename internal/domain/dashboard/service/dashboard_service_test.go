package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/dashboard"
)

type fakeRepo struct {
	start, end time.Time
	err        error
}

func (f *fakeRepo) Totals(_ context.Context, start, end time.Time) (dashboard.Totals, error) {
	f.start, f.end = start, end
	return dashboard.Totals{PickupRevenueCents: 50000, PickupCount: 2, PaidCents: 30000, PaidCount: 1, RentedToday: 1, FleetSize: 2}, f.err
}

func TestDashboardService(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{}
	svc := NewDashboardService(repo)
	svc.now = func() time.Time { return time.Date(2026, 11, 6, 15, 0, 0, 0, time.UTC) }

	s, err := svc.Summary(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "2026-11", s.Month)
	assert.Equal(t, int64(50000), s.PickupRevenueCents)
	assert.Equal(t, int64(30000), s.PaidCents)
	assert.Equal(t, 50.0, s.OccupancyPercent)
	assert.Equal(t, 11, int(repo.start.Month()))

	_, err = svc.Summary(ctx, "13")
	assert.ErrorIs(t, err, dashboard.ErrInvalidMonth)
	repo.err = errors.New("db")
	_, err = svc.Summary(ctx, "2026-10")
	assert.Error(t, err)
}
