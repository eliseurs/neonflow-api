package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/checklist"
)

type fakeRepo struct {
	key string
	req checklist.ItemRequest
	id  string
}

func (f *fakeRepo) List(context.Context) ([]checklist.Item, error) {
	return []checklist.Item{{Key: "pneus"}}, nil
}

func (f *fakeRepo) Create(_ context.Context, key string, req checklist.ItemRequest) (checklist.Item, error) {
	f.key, f.req = key, req
	return checklist.Item{Key: key, Label: req.Label}, nil
}

func (f *fakeRepo) Update(_ context.Context, id string, req checklist.ItemRequest) (checklist.Item, error) {
	f.id, f.req = id, req
	return checklist.Item{ID: id, Label: req.Label}, nil
}

func TestChecklistService(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{}
	svc := NewChecklistService(repo)

	items, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, items, 1)

	it, err := svc.Create(ctx, checklist.ItemRequest{Label: " Calota dianteira ", Position: 110})
	require.NoError(t, err)
	assert.Equal(t, "calota_dianteira", it.Key)
	assert.Equal(t, "Calota dianteira", repo.req.Label)
	assert.Equal(t, 110, repo.req.Position)

	_, err = svc.Create(ctx, checklist.ItemRequest{Label: " "})
	assert.ErrorIs(t, err, checklist.ErrLabelRequired)
	_, err = svc.Create(ctx, checklist.ItemRequest{Label: "Delivery"})
	assert.ErrorIs(t, err, checklist.ErrKeyReserved)

	off := false
	upd, err := svc.Update(ctx, "i1", checklist.ItemRequest{Label: " Pneus e estepe ", Active: &off})
	require.NoError(t, err)
	assert.Equal(t, "i1", repo.id)
	assert.Equal(t, "Pneus e estepe", upd.Label)
	assert.False(t, *repo.req.Active)
	_, err = svc.Update(ctx, "i1", checklist.ItemRequest{})
	assert.ErrorIs(t, err, checklist.ErrLabelRequired)
}
