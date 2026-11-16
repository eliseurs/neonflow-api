//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/account"
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/testutil"
)

// confirmErr é a confirmação quando o teste só olha o erro.
func confirmErr(repo *AccountRepository, tokenHash string) error {
	_, err := repo.ConfirmEmail(context.Background(), tokenHash)
	return err
}

func saveToken(t *testing.T, repo *AccountRepository, tok account.Token) string {
	t.Helper()
	token, hash, err := account.NewToken()
	require.NoError(t, err)
	tok.Hash = hash
	if tok.ExpiresAt.IsZero() {
		tok.ExpiresAt = time.Now().Add(time.Hour)
	}
	require.NoError(t, repo.SaveToken(context.Background(), tok))
	return token
}

func TestAccountRepository_CustomerSignup(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewAccountRepository(database)
	ctx := context.Background()

	// Cadastro feito pela central com o mesmo e-mail, com uma reserva antiga.
	var existing string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO customers (name, email) VALUES ('Ana da Central', 'Ana@neonflow.test') RETURNING id`).Scan(&existing))

	ana, err := repo.CreateCustomerAccount(ctx, account.NewCustomerAccount{Name: "Ana Teste", Email: "ana@neonflow.test", PasswordHash: "h1"})
	require.NoError(t, err)
	assert.Equal(t, "cliente", ana.Role)
	_, err = repo.CreateCustomerAccount(ctx, account.NewCustomerAccount{Name: "Outra", Email: "ANA@neonflow.test", PasswordHash: "h"})
	assert.ErrorIs(t, err, authentication.ErrEmailTaken)

	found, err := repo.FindAccount(ctx, " Ana@NeonFlow.test")
	require.NoError(t, err)
	assert.Equal(t, ana.ID, found.ID)
	assert.False(t, found.Verified)
	_, err = repo.FindAccount(ctx, "ninguem@neonflow.test")
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)

	// Link vencido e link de outra finalidade não confirmam.
	expired := saveToken(t, repo, account.Token{Purpose: account.PurposeConfirmEmail, UserID: ana.ID, ExpiresAt: time.Now().Add(-time.Minute)})
	assert.ErrorIs(t, confirmErr(repo, account.HashToken(expired)), account.ErrInvalidToken)
	reset := saveToken(t, repo, account.Token{Purpose: account.PurposeResetPassword, UserID: ana.ID})
	assert.ErrorIs(t, confirmErr(repo, account.HashToken(reset)), account.ErrInvalidToken)

	// Confirmar liga a conta ao cadastro da central com o mesmo e-mail.
	confirm := saveToken(t, repo, account.Token{Purpose: account.PurposeConfirmEmail, UserID: ana.ID})
	require.NoError(t, confirmErr(repo, account.HashToken(confirm)))
	var customerID sql.NullString
	var verified bool
	require.NoError(t, database.QueryRowContext(ctx, `SELECT customer_id, email_verified_at IS NOT NULL FROM users WHERE id = $1`, ana.ID).Scan(&customerID, &verified))
	assert.True(t, verified)
	assert.Equal(t, existing, customerID.String)
	assert.ErrorIs(t, confirmErr(repo, account.HashToken(confirm)), account.ErrInvalidToken, "o link vale uma vez")

	// Sem cadastro com o e-mail: cria um com o nome e o e-mail da conta.
	bia, err := repo.CreateCustomerAccount(ctx, account.NewCustomerAccount{Name: "Bia Teste", Email: "bia@neonflow.test", PasswordHash: "h2"})
	require.NoError(t, err)
	confirm = saveToken(t, repo, account.Token{Purpose: account.PurposeConfirmEmail, UserID: bia.ID})
	require.NoError(t, confirmErr(repo, account.HashToken(confirm)))
	var name, email string
	require.NoError(t, database.QueryRowContext(ctx,
		`SELECT c.name, c.email FROM users u JOIN customers c ON c.id = u.customer_id WHERE u.id = $1`, bia.ID).Scan(&name, &email))
	assert.Equal(t, "Bia Teste", name)
	assert.Equal(t, "bia@neonflow.test", email)

	// Dois cadastros com o mesmo e-mail: não escolhe, cria um novo.
	for _, n := range []string{"Caio 1", "Caio 2"} {
		_, err = database.ExecContext(ctx, `INSERT INTO customers (name, email) VALUES ($1, 'caio@neonflow.test')`, n)
		require.NoError(t, err)
	}
	caio, err := repo.CreateCustomerAccount(ctx, account.NewCustomerAccount{Name: "Caio Teste", Email: "caio@neonflow.test", PasswordHash: "h3"})
	require.NoError(t, err)
	confirm = saveToken(t, repo, account.Token{Purpose: account.PurposeConfirmEmail, UserID: caio.ID})
	require.NoError(t, confirmErr(repo, account.HashToken(confirm)))
	require.NoError(t, database.QueryRowContext(ctx,
		`SELECT c.name FROM users u JOIN customers c ON c.id = u.customer_id WHERE u.id = $1`, caio.ID).Scan(&name))
	assert.Equal(t, "Caio Teste", name)

	// Nova senha: troca o hash e fecha os outros links de nova senha.
	other := saveToken(t, repo, account.Token{Purpose: account.PurposeResetPassword, UserID: ana.ID})
	require.NoError(t, repo.ResetPassword(ctx, account.HashToken(reset), "h-nova"))
	var hash string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = $1`, ana.ID).Scan(&hash))
	assert.Equal(t, "h-nova", hash)
	assert.ErrorIs(t, repo.ResetPassword(ctx, account.HashToken(other), "h-outra"), account.ErrInvalidToken)

	// Conta desativada não troca a senha.
	_, err = database.ExecContext(ctx, `UPDATE users SET active = false WHERE id = $1`, bia.ID)
	require.NoError(t, err)
	off := saveToken(t, repo, account.Token{Purpose: account.PurposeResetPassword, UserID: bia.ID})
	assert.ErrorIs(t, repo.ResetPassword(ctx, account.HashToken(off), "h"), account.ErrInvalidToken)
	_, err = repo.FindAccount(ctx, "bia@neonflow.test")
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)

	// O banco guarda só o hash.
	var n int
	require.NoError(t, database.QueryRowContext(ctx, `SELECT count(*) FROM account_tokens WHERE token_hash = $1`, confirm).Scan(&n))
	assert.Zero(t, n)
}

func TestAccountRepository_PartnerInvite(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewAccountRepository(database)
	ctx := context.Background()

	var pedro string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO partners (name, is_driver, email, referral_code) VALUES ('Pedro Teste', true, ' Pedro@neonflow.test ', 'PEDRO2') RETURNING id`).Scan(&pedro))

	p, err := repo.PartnerForInvite(ctx, pedro)
	require.NoError(t, err)
	assert.Equal(t, "Pedro Teste", p.Name)
	assert.True(t, p.Active)
	assert.False(t, p.HasAccount)
	assert.False(t, p.EmailTaken)
	_, err = repo.PartnerForInvite(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, account.ErrPartnerNotFound)
	_, err = repo.PartnerForInvite(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, account.ErrPartnerNotFound)

	invite := saveToken(t, repo, account.Token{Purpose: account.PurposeInvite, PartnerID: pedro, ExpiresAt: time.Now().Add(account.InviteTTL)})
	again := saveToken(t, repo, account.Token{Purpose: account.PurposeInvite, PartnerID: pedro})
	require.NoError(t, repo.AcceptInvite(ctx, account.HashToken(invite), "h-pedro"))

	var role, email, hash string
	var partnerID string
	var verified bool
	require.NoError(t, database.QueryRowContext(ctx,
		`SELECT role, email, password_hash, partner_id, email_verified_at IS NOT NULL FROM users WHERE partner_id = $1`, pedro).
		Scan(&role, &email, &hash, &partnerID, &verified))
	assert.Equal(t, "parceiro", role)
	assert.Equal(t, "pedro@neonflow.test", email, "e-mail do cadastro, normalizado")
	assert.Equal(t, "h-pedro", hash)
	assert.True(t, verified, "o convite chegou no e-mail, então ele está confirmado")

	assert.ErrorIs(t, repo.AcceptInvite(ctx, account.HashToken(invite), "h"), account.ErrInvalidToken, "o convite vale uma vez")
	assert.ErrorIs(t, repo.AcceptInvite(ctx, account.HashToken(again), "h"), account.ErrPartnerHasAccount)
	p, err = repo.PartnerForInvite(ctx, pedro)
	require.NoError(t, err)
	assert.True(t, p.HasAccount)

	// E-mail do parceiro já usado por outra conta.
	var bia string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO partners (name, is_referrer, email, referral_code) VALUES ('Bia Hotel', true, 'central@neonflow.test', 'BIAHT2') RETURNING id`).Scan(&bia))
	_, err = database.ExecContext(ctx, `INSERT INTO users (name, email, password_hash, role) VALUES ('Central', 'central@neonflow.test', 'h', 'central')`)
	require.NoError(t, err)
	p, err = repo.PartnerForInvite(ctx, bia)
	require.NoError(t, err)
	assert.True(t, p.EmailTaken)
	taken := saveToken(t, repo, account.Token{Purpose: account.PurposeInvite, PartnerID: bia})
	assert.ErrorIs(t, repo.AcceptInvite(ctx, account.HashToken(taken), "h"), authentication.ErrEmailTaken)

	// Parceiro desativado depois do convite: o link não vale.
	_, err = database.ExecContext(ctx, `UPDATE users SET email = 'outra@neonflow.test' WHERE email = 'central@neonflow.test'`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `UPDATE partners SET active = false WHERE id = $1`, bia)
	require.NoError(t, err)
	late := saveToken(t, repo, account.Token{Purpose: account.PurposeInvite, PartnerID: bia})
	assert.ErrorIs(t, repo.AcceptInvite(ctx, account.HashToken(late), "h"), account.ErrInvalidToken)

	// O banco recusa link sem destino e convite apontando para conta.
	assert.Error(t, repo.SaveToken(ctx, account.Token{Purpose: account.PurposeInvite, Hash: "x1", ExpiresAt: time.Now()}))
	assert.Error(t, repo.SaveToken(ctx, account.Token{Purpose: account.PurposeResetPassword, Hash: "x2", PartnerID: pedro, ExpiresAt: time.Now()}))

	require.NoError(t, database.Close())
	assert.Error(t, repo.AcceptInvite(ctx, account.HashToken(again), "h"))
	assert.Error(t, confirmErr(repo, "x"))
	assert.Error(t, repo.ResetPassword(ctx, "x", "h"))
	_, err = repo.FindAccount(ctx, "a@neonflow.test")
	assert.Error(t, err)
	_, err = repo.CreateCustomerAccount(ctx, account.NewCustomerAccount{Name: "X", Email: "x@neonflow.test", PasswordHash: "h"})
	assert.Error(t, err)
	_, err = repo.PartnerForInvite(ctx, pedro)
	assert.Error(t, err)
}
