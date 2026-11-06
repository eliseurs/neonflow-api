//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/user"
	"neonflow-api/internal/testutil"
)

func TestUserRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewUserRepository(database)
	ctx := context.Background()

	ana, err := repo.Create(ctx, authentication.NewUser{Name: "Ana Teste", Email: "ana@neonflow.test", PasswordHash: "h1", Role: "central"})
	require.NoError(t, err)
	assert.True(t, ana.Active)
	_, err = repo.Create(ctx, authentication.NewUser{Name: "Outra", Email: "ANA@neonflow.test", PasswordHash: "h", Role: "admin"})
	assert.ErrorIs(t, err, authentication.ErrEmailTaken)
	bia, err := repo.Create(ctx, authentication.NewUser{Name: "Bia Teste", Email: "bia@neonflow.test", PasswordHash: "h2", Role: "admin"})
	require.NoError(t, err)

	off := false
	updated, err := repo.Update(ctx, ana.ID, user.UpdateRequest{Name: "Ana Souza Teste", Role: "admin", Active: &off})
	require.NoError(t, err)
	assert.Equal(t, "Ana Souza Teste", updated.Name)
	assert.Equal(t, "admin", updated.Role)
	assert.False(t, updated.Active)
	kept, err := repo.Update(ctx, ana.ID, user.UpdateRequest{Name: "Ana Souza Teste", Role: "admin"})
	require.NoError(t, err)
	assert.False(t, kept.Active, "sem active, mantém")

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, bia.ID, list[0].ID, "ativos primeiro")

	hash, err := repo.PasswordHash(ctx, bia.ID)
	require.NoError(t, err)
	assert.Equal(t, "h2", hash)
	require.NoError(t, repo.SetPassword(ctx, bia.ID, "h3"))
	hash, err = repo.PasswordHash(ctx, bia.ID)
	require.NoError(t, err)
	assert.Equal(t, "h3", hash)

	missing := "00000000-0000-0000-0000-000000000000"
	_, err = repo.FindByID(ctx, missing)
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)
	_, err = repo.FindByID(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)
	_, err = repo.Update(ctx, missing, user.UpdateRequest{Name: "X", Role: "central"})
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)
	_, err = repo.PasswordHash(ctx, missing)
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)
	assert.ErrorIs(t, repo.SetPassword(ctx, missing, "h"), authentication.ErrUserNotFound)

	require.NoError(t, database.Close())
	_, err = repo.List(ctx)
	assert.Error(t, err)
	assert.Error(t, repo.SetPassword(ctx, bia.ID, "h"))
}
