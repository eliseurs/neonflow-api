//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/partner"
	"neonflow-api/internal/testutil"
)

func TestPartnerRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewPartnerRepository(database)
	ctx := context.Background()
	pct := func(v float64) *float64 { return &v }

	hotel, err := repo.Create(ctx, "HTL234", partner.Request{Name: "Hotel Teste", Kinds: []string{partner.KindReferrer},
		WhatsApp: "5554999990000", City: "Gramado", CommissionPercent: pct(10)})
	require.NoError(t, err)
	assert.True(t, hotel.Active, "ativo por padrão")
	assert.Equal(t, []string{partner.KindReferrer}, hotel.Kinds)
	assert.Equal(t, 10.0, *hotel.CommissionPercent)
	assert.Nil(t, hotel.OwnerSharePercent)

	owner, err := repo.Create(ctx, "OWN234", partner.Request{Name: "Dono Teste",
		Kinds: []string{partner.KindOwner, partner.KindDriver, partner.KindReferrer}, OwnerSharePercent: pct(70.5)})
	require.NoError(t, err)
	assert.Equal(t, []string{partner.KindOwner, partner.KindDriver, partner.KindReferrer}, owner.Kinds)
	assert.Equal(t, 70.5, *owner.OwnerSharePercent)

	_, err = repo.Create(ctx, "HTL234", partner.Request{Name: "Repetido", Kinds: []string{partner.KindReferrer}})
	assert.ErrorIs(t, err, partner.ErrCodeTaken)
	_, err = repo.Create(ctx, "NEW234", partner.Request{Name: "Sem tipo"})
	assert.Error(t, err, "CHECK: ao menos um tipo")

	off := false
	updated, err := repo.Update(ctx, hotel.ID, partner.Request{Name: "Hotel Novo", Kinds: []string{partner.KindReferrer}, Active: &off})
	require.NoError(t, err)
	assert.Equal(t, "HTL234", updated.ReferralCode, "o código não muda")
	assert.False(t, updated.Active)
	assert.Nil(t, updated.CommissionPercent, "PUT substitui")

	owners, err := repo.List(ctx, partner.KindOwner)
	require.NoError(t, err)
	require.Len(t, owners, 1)
	assert.Equal(t, owner.ID, owners[0].ID)
	all, err := repo.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, owner.ID, all[0].ID, "ativos primeiro")

	byCode, err := repo.FindByCode(ctx, "OWN234")
	require.NoError(t, err)
	assert.Equal(t, owner.ID, byCode.ID)
	got, err := repo.FindByID(ctx, hotel.ID)
	require.NoError(t, err)
	assert.Equal(t, "Hotel Novo", got.Name)

	missing := "00000000-0000-0000-0000-000000000000"
	_, err = repo.FindByID(ctx, missing)
	assert.ErrorIs(t, err, partner.ErrPartnerNotFound)
	_, err = repo.FindByID(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, partner.ErrPartnerNotFound)
	_, err = repo.FindByCode(ctx, "XXX999")
	assert.ErrorIs(t, err, partner.ErrPartnerNotFound)
	_, err = repo.Update(ctx, missing, partner.Request{Name: "X", Kinds: []string{partner.KindReferrer}})
	assert.ErrorIs(t, err, partner.ErrPartnerNotFound)

	require.NoError(t, database.Close())
	_, err = repo.List(ctx, "")
	assert.Error(t, err)
}
