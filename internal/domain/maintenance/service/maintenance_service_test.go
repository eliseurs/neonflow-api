package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/maintenance"
)

type fakeRepo struct {
	saved   maintenance.Request
	user    string
	deleted string
}

func (f *fakeRepo) List(context.Context, string) ([]maintenance.Record, error) {
	return []maintenance.Record{{ID: "m1"}}, nil
}
func (f *fakeRepo) Create(_ context.Context, _, user string, r maintenance.Request) (maintenance.Record, error) {
	f.saved, f.user = r, user
	return maintenance.Record{ID: "m1", Kind: r.Kind}, nil
}
func (f *fakeRepo) Update(_ context.Context, id string, r maintenance.Request) (maintenance.Record, error) {
	f.saved = r
	return maintenance.Record{ID: id, Kind: r.Kind}, nil
}
func (f *fakeRepo) Delete(_ context.Context, id string) error {
	f.deleted = id
	return nil
}

func TestMaintenanceService(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{}
	svc := NewMaintenanceService(repo)
	svc.now = func() time.Time { return time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC) }

	list, err := svc.List(ctx, "v1")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	rec, err := svc.Create(ctx, "v1", "u1", maintenance.Request{Date: "2026-11-01", Kind: "Revisao", Description: "Revisão 60 mil"})
	require.NoError(t, err)
	assert.Equal(t, "revisao", rec.Kind)
	assert.Equal(t, "u1", repo.user)
	_, err = svc.Create(ctx, "v1", "u1", maintenance.Request{Date: "2026-12-01", Kind: "outros", Description: "x"})
	assert.ErrorIs(t, err, maintenance.ErrFutureDate)
	_, err = svc.Update(ctx, "m1", maintenance.Request{Date: "2026-11-01", Kind: "pneus", Description: "4 pneus"})
	require.NoError(t, err)
	_, err = svc.Update(ctx, "m1", maintenance.Request{Date: "x", Kind: "pneus", Description: "4 pneus"})
	assert.ErrorIs(t, err, maintenance.ErrInvalidDate)
	require.NoError(t, svc.Delete(ctx, "m1"))
	assert.Equal(t, "m1", repo.deleted)
}
