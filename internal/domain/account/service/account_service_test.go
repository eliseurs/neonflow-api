package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/account"
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/infra/email"
)

type fakeRepo struct {
	createErr error
	existing  account.Account
	findErr   error
	tokens    []account.Token
	tokenErr  error
	partner   account.PartnerContact
	partnerEr error

	confirmed, reset, accepted string
	passwordHash               string
	linkErr                    error
}

func (f *fakeRepo) CreateCustomerAccount(_ context.Context, n account.NewCustomerAccount) (account.Account, error) {
	if f.createErr != nil {
		return account.Account{}, f.createErr
	}
	f.passwordHash = n.PasswordHash
	return account.Account{ID: "u1", Name: n.Name, Email: n.Email, Role: "cliente"}, nil
}

func (f *fakeRepo) FindAccount(context.Context, string) (account.Account, error) {
	return f.existing, f.findErr
}

func (f *fakeRepo) SaveToken(_ context.Context, t account.Token) error {
	f.tokens = append(f.tokens, t)
	return f.tokenErr
}

func (f *fakeRepo) ConfirmEmail(_ context.Context, hash string) error {
	f.confirmed = hash
	return f.linkErr
}

func (f *fakeRepo) ResetPassword(_ context.Context, hash, pw string) error {
	f.reset, f.passwordHash = hash, pw
	return f.linkErr
}

func (f *fakeRepo) PartnerForInvite(context.Context, string) (account.PartnerContact, error) {
	return f.partner, f.partnerEr
}

func (f *fakeRepo) AcceptInvite(_ context.Context, hash, pw string) error {
	f.accepted, f.passwordHash = hash, pw
	return f.linkErr
}

type fakeMail struct {
	off  bool
	err  error
	sent []email.Message
}

func (m *fakeMail) Enabled() bool { return !m.off }
func (m *fakeMail) Send(_ context.Context, msg email.Message) error {
	m.sent = append(m.sent, msg)
	return m.err
}

type fakeHasher struct{ err error }

func (h fakeHasher) Hash(p string) (string, error) { return "h:" + p, h.err }

type spyLog struct{ args []any }

func (s *spyLog) Error(_ string, args ...any) { s.args = append(s.args, args...) }

var now = time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)

func newSvc(repo *fakeRepo, mail *fakeMail) (*AccountService, *spyLog) {
	log := &spyLog{}
	s := NewAccountService(repo, fakeHasher{}, mail, log, "https://app.neonflow.test")
	s.now = func() time.Time { return now }
	return s, log
}

// tokenIn confere que o código do link no e-mail é o do hash gravado.
func tokenIn(t *testing.T, msg email.Message, saved account.Token, prefix string) {
	t.Helper()
	assert.Contains(t, msg.Text, prefix)
	i := len(prefix) + strings.Index(msg.Text, prefix)
	token := msg.Text[i : i+43]
	assert.Equal(t, saved.Hash, account.HashToken(token), "o link do e-mail é o do hash gravado")
}

var signup = account.SignupRequest{Name: "Ana Teste", Email: "ANA@neonflow.test", Password: "senha-forte-1"}

func TestSignup_NewAccount(t *testing.T) {
	repo, mail := &fakeRepo{}, &fakeMail{}
	svc, _ := newSvc(repo, mail)
	require.NoError(t, svc.Signup(context.Background(), signup))

	assert.Equal(t, "h:senha-forte-1", repo.passwordHash, "grava só o hash")
	require.Len(t, repo.tokens, 1)
	assert.Equal(t, account.PurposeConfirmEmail, repo.tokens[0].Purpose)
	assert.Equal(t, "u1", repo.tokens[0].UserID)
	assert.Equal(t, now.Add(24*time.Hour), repo.tokens[0].ExpiresAt)
	require.Len(t, mail.sent, 1)
	assert.Equal(t, "ana@neonflow.test", mail.sent[0].To)
	tokenIn(t, mail.sent[0], repo.tokens[0], "https://app.neonflow.test/conta/confirmar/")
}

func TestSignup_ExistingEmail(t *testing.T) {
	taken := authentication.ErrEmailTaken

	// Cliente ainda sem confirmar: reenvia o link.
	repo := &fakeRepo{createErr: taken, existing: account.Account{ID: "u9", Name: "Ana", Email: "ana@neonflow.test", Role: "cliente"}}
	mail := &fakeMail{}
	svc, _ := newSvc(repo, mail)
	require.NoError(t, svc.Signup(context.Background(), signup))
	require.Len(t, repo.tokens, 1)
	assert.Equal(t, "u9", repo.tokens[0].UserID)
	assert.Len(t, mail.sent, 1)

	// Conta confirmada, da equipe ou desativada: mesma resposta, nada enviado.
	for name, repo := range map[string]*fakeRepo{
		"cliente confirmado": {createErr: taken, existing: account.Account{ID: "u9", Role: "cliente", Verified: true}},
		"equipe":             {createErr: taken, existing: account.Account{ID: "u9", Role: "central", Verified: true}},
		"desativada":         {createErr: taken, findErr: authentication.ErrUserNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			mail := &fakeMail{}
			svc, _ := newSvc(repo, mail)
			require.NoError(t, svc.Signup(context.Background(), signup))
			assert.Empty(t, repo.tokens)
			assert.Empty(t, mail.sent)
		})
	}

	boom := errors.New("db")
	svc, _ = newSvc(&fakeRepo{createErr: taken, findErr: boom}, &fakeMail{})
	assert.ErrorIs(t, svc.Signup(context.Background(), signup), boom)
}

func TestSignup_Errors(t *testing.T) {
	svc, _ := newSvc(&fakeRepo{}, &fakeMail{off: true})
	assert.ErrorIs(t, svc.Signup(context.Background(), signup), email.ErrNotConfigured, "sem envio não cria conta que não pode confirmar")

	svc, _ = newSvc(&fakeRepo{}, &fakeMail{})
	assert.ErrorIs(t, svc.Signup(context.Background(), account.SignupRequest{Name: "Ana", Email: "x", Password: "senha-forte-1"}), authentication.ErrInvalidEmail)

	boom := errors.New("db")
	svc, _ = newSvc(&fakeRepo{createErr: boom}, &fakeMail{})
	assert.ErrorIs(t, svc.Signup(context.Background(), signup), boom)

	svc, _ = newSvc(&fakeRepo{tokenErr: boom}, &fakeMail{})
	assert.ErrorIs(t, svc.Signup(context.Background(), signup), boom)

	svc = NewAccountService(&fakeRepo{}, fakeHasher{err: boom}, &fakeMail{}, &spyLog{}, "")
	assert.ErrorIs(t, svc.Signup(context.Background(), signup), boom)

	svc, log := newSvc(&fakeRepo{}, &fakeMail{err: errors.New("resend: 403")})
	assert.ErrorIs(t, svc.Signup(context.Background(), signup), account.ErrSendFailed)
	assert.NotContains(t, log.args, "ana@neonflow.test", "o log não leva o destinatário")
}

func TestConfirmEmail(t *testing.T) {
	repo := &fakeRepo{}
	svc, _ := newSvc(repo, &fakeMail{})
	require.NoError(t, svc.ConfirmEmail(context.Background(), account.TokenRequest{Token: "abc"}))
	assert.Equal(t, account.HashToken("abc"), repo.confirmed)

	assert.ErrorIs(t, svc.ConfirmEmail(context.Background(), account.TokenRequest{}), account.ErrInvalidToken)
	svc, _ = newSvc(&fakeRepo{linkErr: account.ErrInvalidToken}, &fakeMail{})
	assert.ErrorIs(t, svc.ConfirmEmail(context.Background(), account.TokenRequest{Token: "abc"}), account.ErrInvalidToken)
}

func TestForgotPassword(t *testing.T) {
	repo := &fakeRepo{existing: account.Account{ID: "u3", Name: "Bia", Email: "bia@neonflow.test", Role: "parceiro", Verified: true}}
	mail := &fakeMail{}
	svc, _ := newSvc(repo, mail)
	require.NoError(t, svc.ForgotPassword(context.Background(), account.EmailRequest{Email: " BIA@neonflow.test"}))
	require.Len(t, repo.tokens, 1)
	assert.Equal(t, account.PurposeResetPassword, repo.tokens[0].Purpose)
	assert.Equal(t, now.Add(time.Hour), repo.tokens[0].ExpiresAt)
	require.Len(t, mail.sent, 1)
	tokenIn(t, mail.sent[0], repo.tokens[0], "https://app.neonflow.test/conta/nova-senha/")

	// Sem conta ou sem e-mail confirmado: mesma resposta, nada enviado.
	for name, repo := range map[string]*fakeRepo{
		"sem conta":      {findErr: authentication.ErrUserNotFound},
		"não confirmado": {existing: account.Account{ID: "u4", Role: "cliente"}},
	} {
		t.Run(name, func(t *testing.T) {
			mail := &fakeMail{}
			svc, _ := newSvc(repo, mail)
			require.NoError(t, svc.ForgotPassword(context.Background(), account.EmailRequest{Email: "x@neonflow.test"}))
			assert.Empty(t, mail.sent)
		})
	}

	svc, _ = newSvc(&fakeRepo{}, &fakeMail{off: true})
	assert.ErrorIs(t, svc.ForgotPassword(context.Background(), account.EmailRequest{Email: "x@neonflow.test"}), email.ErrNotConfigured)
	svc, _ = newSvc(&fakeRepo{}, &fakeMail{})
	assert.ErrorIs(t, svc.ForgotPassword(context.Background(), account.EmailRequest{Email: "nao-e-email"}), authentication.ErrInvalidEmail)
	boom := errors.New("db")
	svc, _ = newSvc(&fakeRepo{findErr: boom}, &fakeMail{})
	assert.ErrorIs(t, svc.ForgotPassword(context.Background(), account.EmailRequest{Email: "x@neonflow.test"}), boom)
	svc, _ = newSvc(&fakeRepo{existing: account.Account{ID: "u3", Verified: true}, tokenErr: boom}, &fakeMail{})
	assert.ErrorIs(t, svc.ForgotPassword(context.Background(), account.EmailRequest{Email: "x@neonflow.test"}), boom)
}

func TestResetPassword(t *testing.T) {
	repo := &fakeRepo{}
	svc, _ := newSvc(repo, &fakeMail{})
	require.NoError(t, svc.ResetPassword(context.Background(), account.PasswordRequest{Token: "abc", Password: "senha-nova-123"}))
	assert.Equal(t, account.HashToken("abc"), repo.reset)
	assert.Equal(t, "h:senha-nova-123", repo.passwordHash)

	assert.ErrorIs(t, svc.ResetPassword(context.Background(), account.PasswordRequest{Password: "senha-nova-123"}), account.ErrInvalidToken)
	assert.ErrorIs(t, svc.ResetPassword(context.Background(), account.PasswordRequest{Token: "abc", Password: "curta"}), authentication.ErrWeakPassword)
	boom := errors.New("hash")
	svc = NewAccountService(&fakeRepo{}, fakeHasher{err: boom}, &fakeMail{}, &spyLog{}, "")
	assert.ErrorIs(t, svc.ResetPassword(context.Background(), account.PasswordRequest{Token: "abc", Password: "senha-nova-123"}), boom)
}

func TestInvitePartner(t *testing.T) {
	repo := &fakeRepo{partner: account.PartnerContact{Name: "Pedro", Email: " Pedro@neonflow.test ", Active: true}}
	mail := &fakeMail{}
	svc, _ := newSvc(repo, mail)
	inv, err := svc.InvitePartner(context.Background(), "p1")
	require.NoError(t, err)
	assert.Equal(t, "pedro@neonflow.test", inv.SentTo)
	assert.Equal(t, now.Add(7*24*time.Hour), inv.ExpiresAt)
	require.Len(t, repo.tokens, 1)
	assert.Equal(t, account.Token{Purpose: account.PurposeInvite, PartnerID: "p1", Hash: repo.tokens[0].Hash, ExpiresAt: inv.ExpiresAt}, repo.tokens[0])
	require.Len(t, mail.sent, 1)
	assert.Equal(t, "pedro@neonflow.test", mail.sent[0].To)
	tokenIn(t, mail.sent[0], repo.tokens[0], "https://app.neonflow.test/conta/convite/")

	cases := map[string]struct {
		partner account.PartnerContact
		err     error
		want    error
	}{
		"inexistente":     {err: account.ErrPartnerNotFound, want: account.ErrPartnerNotFound},
		"desativado":      {partner: account.PartnerContact{Email: "p@neonflow.test"}, want: account.ErrPartnerInactive},
		"já tem conta":    {partner: account.PartnerContact{Email: "p@neonflow.test", Active: true, HasAccount: true}, want: account.ErrPartnerHasAccount},
		"sem e-mail":      {partner: account.PartnerContact{Active: true}, want: account.ErrPartnerWithoutEmail},
		"e-mail inválido": {partner: account.PartnerContact{Email: "pedro", Active: true}, want: account.ErrPartnerWithoutEmail},
		"e-mail em uso":   {partner: account.PartnerContact{Email: "p@neonflow.test", Active: true, EmailTaken: true}, want: authentication.ErrEmailTaken},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &fakeRepo{partner: tc.partner, partnerEr: tc.err}
			mail := &fakeMail{}
			svc, _ := newSvc(repo, mail)
			_, err := svc.InvitePartner(context.Background(), "p1")
			assert.ErrorIs(t, err, tc.want)
			assert.Empty(t, repo.tokens)
			assert.Empty(t, mail.sent)
		})
	}

	svc, _ = newSvc(&fakeRepo{}, &fakeMail{off: true})
	_, err = svc.InvitePartner(context.Background(), "p1")
	assert.ErrorIs(t, err, email.ErrNotConfigured)
	boom := errors.New("db")
	svc, _ = newSvc(&fakeRepo{partner: account.PartnerContact{Email: "p@neonflow.test", Active: true}, tokenErr: boom}, &fakeMail{})
	_, err = svc.InvitePartner(context.Background(), "p1")
	assert.ErrorIs(t, err, boom)
	svc, _ = newSvc(&fakeRepo{partner: account.PartnerContact{Email: "p@neonflow.test", Active: true}}, &fakeMail{err: boom})
	_, err = svc.InvitePartner(context.Background(), "p1")
	assert.ErrorIs(t, err, account.ErrSendFailed)
}

func TestAcceptInvite(t *testing.T) {
	repo := &fakeRepo{}
	svc, _ := newSvc(repo, &fakeMail{})
	require.NoError(t, svc.AcceptInvite(context.Background(), account.PasswordRequest{Token: "abc", Password: "senha-forte-1"}))
	assert.Equal(t, account.HashToken("abc"), repo.accepted)
	assert.Equal(t, "h:senha-forte-1", repo.passwordHash)

	assert.ErrorIs(t, svc.AcceptInvite(context.Background(), account.PasswordRequest{Token: "abc"}), authentication.ErrWeakPassword)
	svc, _ = newSvc(&fakeRepo{linkErr: account.ErrPartnerHasAccount}, &fakeMail{})
	assert.ErrorIs(t, svc.AcceptInvite(context.Background(), account.PasswordRequest{Token: "abc", Password: "senha-forte-1"}), account.ErrPartnerHasAccount)
}
