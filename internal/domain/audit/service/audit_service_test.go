package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"neonflow-api/internal/domain/audit"
)

type fakeRepo struct{ got audit.Filter }

func (f *fakeRepo) List(_ context.Context, filter audit.Filter) ([]audit.Entry, error) {
	f.got = filter
	return []audit.Entry{}, nil
}

func TestList(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewAuditService(repo)

	_, err := svc.List(context.Background(), audit.Filter{UserID: "u1"})
	assert.NoError(t, err)
	assert.Equal(t, audit.Filter{UserID: "u1", Limit: audit.DefaultLimit}, repo.got, "sem limite usa o padrão")

	_, err = svc.List(context.Background(), audit.Filter{Limit: 20, Denied: true})
	assert.NoError(t, err)
	assert.Equal(t, 20, repo.got.Limit)

	for _, l := range []int{-1, audit.MaxLimit + 1} {
		_, err = svc.List(context.Background(), audit.Filter{Limit: l})
		assert.ErrorIs(t, err, audit.ErrInvalidLimit)
	}
}
