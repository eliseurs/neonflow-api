//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/checklist"
	"neonflow-api/internal/testutil"
)

func TestChecklistRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewChecklistRepository(database)
	ctx := context.Background()

	// Lista inicial do fundador (RN-01.6), ativa e em ordem.
	items, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, items, 10)
	assert.Equal(t, "combustivel", items[0].Key)
	assert.Equal(t, "Combustível", items[0].Label)
	assert.Equal(t, "quilometragem", items[9].Key)
	for _, it := range items {
		assert.True(t, it.Active, it.Key)
	}

	created, err := repo.Create(ctx, "calota", checklist.ItemRequest{Label: "Calota", Position: 105})
	require.NoError(t, err)
	assert.True(t, created.Active, "ativo por padrão")
	assert.Equal(t, 105, created.Position)

	_, err = repo.Create(ctx, "calota", checklist.ItemRequest{Label: "Calota de novo"})
	assert.ErrorIs(t, err, checklist.ErrKeyTaken)

	off := false
	updated, err := repo.Update(ctx, created.ID, checklist.ItemRequest{Label: "Calotas", Position: 5, Active: &off})
	require.NoError(t, err)
	assert.Equal(t, "calota", updated.Key, "a chave não muda")
	assert.Equal(t, "Calotas", updated.Label)
	assert.False(t, updated.Active)

	// Sem "active", mantém o valor atual.
	kept, err := repo.Update(ctx, created.ID, checklist.ItemRequest{Label: "Calotas", Position: 5})
	require.NoError(t, err)
	assert.False(t, kept.Active)

	items, err = repo.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, "calota", items[0].Key, "ordenado pela posição")

	_, err = repo.Update(ctx, "00000000-0000-0000-0000-000000000000", checklist.ItemRequest{Label: "x"})
	assert.ErrorIs(t, err, checklist.ErrItemNotFound)
	_, err = repo.Update(ctx, "nao-e-uuid", checklist.ItemRequest{Label: "x"})
	assert.ErrorIs(t, err, checklist.ErrItemNotFound)

	require.NoError(t, database.Close())
	_, err = repo.List(ctx)
	assert.Error(t, err)
}
