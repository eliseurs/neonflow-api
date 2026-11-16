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
	assert.True(t, creds.EmailVerified, "a equipe nasce com o e-mail confirmado")
	assert.Nil(t, creds.PartnerID)

	// Conta de parceiro: traz o cadastro e some quando o parceiro é desativado (RN-05.12).
	var partnerID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO partners (name, is_owner, referral_code) VALUES ('Dono Teste', true, 'DONO23') RETURNING id`).Scan(&partnerID))
	_, err = database.ExecContext(ctx,
		`INSERT INTO users (name, email, password_hash, role, partner_id, email_verified_at) VALUES ('Dono Teste', 'dono@neonflow.test', 'h', 'parceiro', $1, now())`, partnerID)
	require.NoError(t, err)
	dono, err := repo.FindByEmail(ctx, "dono@neonflow.test")
	require.NoError(t, err)
	require.NotNil(t, dono.PartnerID)
	assert.Equal(t, partnerID, *dono.PartnerID)
	me, err := repo.FindByID(ctx, dono.ID)
	require.NoError(t, err)
	assert.Equal(t, partnerID, *me.PartnerID)
	_, err = database.ExecContext(ctx, `UPDATE partners SET active = false WHERE id = $1`, partnerID)
	require.NoError(t, err)
	_, err = repo.FindByEmail(ctx, "dono@neonflow.test")
	assert.ErrorIs(t, err, authentication.ErrUserNotFound, "parceiro desativado não entra")

	// O banco recusa conta de parceiro sem parceiro e conta da equipe ligada a um cadastro.
	_, err = database.ExecContext(ctx, `INSERT INTO users (name, email, password_hash, role) VALUES ('X', 'p@neonflow.test', 'h', 'parceiro')`)
	assert.Error(t, err)
	_, err = database.ExecContext(ctx, `INSERT INTO users (name, email, password_hash, role, partner_id) VALUES ('X', 'c@neonflow.test', 'h', 'central', $1)`, partnerID)
	assert.Error(t, err)

	// Cliente sem o e-mail confirmado: o repository devolve, o serviço recusa o login.
	_, err = database.ExecContext(ctx, `INSERT INTO users (name, email, password_hash, role) VALUES ('Ana', 'ana@neonflow.test', 'h', 'cliente')`)
	require.NoError(t, err)
	ana, err := repo.FindByEmail(ctx, "ana@neonflow.test")
	require.NoError(t, err)
	assert.False(t, ana.EmailVerified)
	assert.Nil(t, ana.CustomerID)

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
