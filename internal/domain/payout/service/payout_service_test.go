package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/payout"
)

type fakeRepo struct {
	filter payout.Filter
	paidBy string
}

func (f *fakeRepo) List(_ context.Context, filter payout.Filter) ([]payout.Payout, error) {
	f.filter = filter
	return []payout.Payout{{ID: "o1"}}, nil
}
func (f *fakeRepo) MarkPaid(_ context.Context, id, userID string) (payout.Payout, error) {
	f.paidBy = userID
	return payout.Payout{ID: id, Status: payout.StatusPaid}, nil
}

func TestPayoutService(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{}
	svc := NewPayoutService(repo)

	list, err := svc.List(ctx, payout.Filter{Kind: payout.KindOwnerShare, Status: payout.StatusPending, PartnerID: "p1"})
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "p1", repo.filter.PartnerID)
	_, err = svc.List(ctx, payout.Filter{Status: "atrasada"})
	assert.ErrorIs(t, err, payout.ErrInvalidFilter)

	p, err := svc.MarkPaid(ctx, "o1", "u1")
	require.NoError(t, err)
	assert.Equal(t, payout.StatusPaid, p.Status)
	assert.Equal(t, "u1", repo.paidBy)
}
