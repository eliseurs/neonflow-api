//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/testutil"
)

func TestAuthRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewAuthRepository(database)
	ctx := context.Background()

	created, err := repo.Create(ctx, authentication.NewUser{Name: "Central Teste", Email: "central@neonflow.test", PasswordHash: "hash", Role: "central"})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.False(t, created.CreatedAt.IsZero())

	_, err = repo.Create(ctx, authentication.NewUser{Name: "Outro", Email: "CENTRAL@neonflow.test", PasswordHash: "h", Role: "admin"})
	assert.ErrorIs(t, err, authentication.ErrEmailTaken, "e-mail único sem diferenciar maiúsculas (RN-05.4)")

	_, err = repo.Create(ctx, authentication.NewUser{Name: "X", Email: "x@neonflow.test", PasswordHash: "h", Role: "motorista"})
	assert.Error(t, err, "CHECK de papel no banco")

	creds, err := repo.FindByEmail(ctx, "  Central@NeonFlow.test ")
	require.NoError(t, err)
	assert.Equal(t, created.ID, creds.ID)
	assert.Equal(t, "hash", creds.PasswordHash)

	_, err = repo.FindByEmail(ctx, "ninguem@neonflow.test")
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)

	u, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Central Teste", u.Name)

	// Desativado não entra nem mantém a sessão.
	_, err = database.ExecContext(ctx, `UPDATE users SET active = false WHERE id = $1`, created.ID)
	require.NoError(t, err)
	_, err = repo.FindByEmail(ctx, "central@neonflow.test")
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)
	_, err = repo.FindByID(ctx, created.ID)
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)

	_, err = repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)

	_, err = repo.FindByID(ctx, "nao-e-uuid")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, authentication.ErrUserNotFound)

	require.NoError(t, database.Close())
	_, err = repo.FindByEmail(ctx, "central@neonflow.test")
	assert.Error(t, err)
	_, err = repo.Create(ctx, authentication.NewUser{Name: "Y", Email: "y@neonflow.test", PasswordHash: "h", Role: "admin"})
	assert.Error(t, err)
}
