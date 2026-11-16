package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"neonflow-api/internal/domain/account"
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/infra/email"
	"neonflow-api/internal/pkg/authz"
)

type AccountService struct {
	repo   account.Repository
	hasher account.PasswordHasher
	mail   account.Mailer
	log    account.Logger
	// appURL é o endereço do neonflow-app, sem a barra final; os links dos e-mails abrem páginas dele.
	appURL string
	now    func() time.Time
}

func NewAccountService(repo account.Repository, hasher account.PasswordHasher, mail account.Mailer, log account.Logger, appURL string) *AccountService {
	return &AccountService{repo: repo, hasher: hasher, mail: mail, log: log, appURL: appURL, now: time.Now}
}

// Signup cria a conta do cliente e manda o link de confirmação (RN-05.13). A resposta é a mesma quando o
// e-mail já tem conta, para o cadastro não revelar quem é cliente; se a conta existente ainda não foi
// confirmada, o link é reenviado.
func (s *AccountService) Signup(ctx context.Context, req account.SignupRequest) error {
	if !s.mail.Enabled() {
		return email.ErrNotConfigured
	}
	valid, err := req.Validate()
	if err != nil {
		return err
	}
	hash, err := s.hasher.Hash(valid.Password)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}

	acc, err := s.repo.CreateCustomerAccount(ctx, account.NewCustomerAccount{Name: valid.Name, Email: valid.Email, PasswordHash: hash})
	if errors.Is(err, authentication.ErrEmailTaken) {
		acc, err = s.repo.FindAccount(ctx, valid.Email)
		if errors.Is(err, authentication.ErrUserNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if acc.Role != authz.RoleCustomer || acc.Verified {
			return nil
		}
	} else if err != nil {
		return err
	}

	token, err := s.issue(ctx, account.Token{Purpose: account.PurposeConfirmEmail, UserID: acc.ID}, account.ConfirmEmailTTL)
	if err != nil {
		return err
	}
	return s.send(ctx, email.Message{
		To:      acc.Email,
		Subject: "Confirme o seu e-mail no Neon Flow",
		Text: fmt.Sprintf("Olá, %s.\n\nPara ativar a sua conta no Neon Flow, abra o link abaixo em até 24 horas:\n\n%s/conta/confirmar/%s\n\n"+
			"Se você não criou esta conta, ignore este e-mail.\n\nNeon Flow", acc.Name, s.appURL, token),
	})
}

func (s *AccountService) ConfirmEmail(ctx context.Context, req account.TokenRequest) error {
	if req.Token == "" {
		return account.ErrInvalidToken
	}
	return s.repo.ConfirmEmail(ctx, account.HashToken(req.Token))
}

// ForgotPassword manda o link de nova senha para qualquer conta ativa e confirmada (RN-05.14). A resposta
// não muda quando o e-mail não tem conta.
func (s *AccountService) ForgotPassword(ctx context.Context, req account.EmailRequest) error {
	if !s.mail.Enabled() {
		return email.ErrNotConfigured
	}
	addr := authentication.NormalizeEmail(req.Email)
	if !account.ValidEmail(addr) {
		return authentication.ErrInvalidEmail
	}
	acc, err := s.repo.FindAccount(ctx, addr)
	if errors.Is(err, authentication.ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !acc.Verified {
		return nil
	}

	token, err := s.issue(ctx, account.Token{Purpose: account.PurposeResetPassword, UserID: acc.ID}, account.ResetPasswordTTL)
	if err != nil {
		return err
	}
	return s.send(ctx, email.Message{
		To:      acc.Email,
		Subject: "Nova senha no Neon Flow",
		Text: fmt.Sprintf("Olá, %s.\n\nPara criar uma senha nova, abra o link abaixo em até 1 hora:\n\n%s/conta/nova-senha/%s\n\n"+
			"Se você não pediu, ignore este e-mail: a senha atual continua valendo.\n\nNeon Flow", acc.Name, s.appURL, token),
	})
}

func (s *AccountService) ResetPassword(ctx context.Context, req account.PasswordRequest) error {
	hash, err := s.passwordFromLink(req)
	if err != nil {
		return err
	}
	return s.repo.ResetPassword(ctx, account.HashToken(req.Token), hash)
}

// InvitePartner manda ao parceiro o link para criar a senha (RN-05.12). Quem convida é a central.
func (s *AccountService) InvitePartner(ctx context.Context, partnerID string) (account.Invite, error) {
	if !s.mail.Enabled() {
		return account.Invite{}, email.ErrNotConfigured
	}
	p, err := s.repo.PartnerForInvite(ctx, partnerID)
	if err != nil {
		return account.Invite{}, err
	}
	p.Email = authentication.NormalizeEmail(p.Email)
	switch {
	case !p.Active:
		return account.Invite{}, account.ErrPartnerInactive
	case p.HasAccount:
		return account.Invite{}, account.ErrPartnerHasAccount
	case !account.ValidEmail(p.Email):
		return account.Invite{}, account.ErrPartnerWithoutEmail
	case p.EmailTaken:
		return account.Invite{}, authentication.ErrEmailTaken
	}

	expiresAt := s.now().Add(account.InviteTTL)
	token, err := s.issue(ctx, account.Token{Purpose: account.PurposeInvite, PartnerID: partnerID}, account.InviteTTL)
	if err != nil {
		return account.Invite{}, err
	}
	err = s.send(ctx, email.Message{
		To:      p.Email,
		Subject: "O seu acesso ao Neon Flow",
		Text: fmt.Sprintf("Olá, %s.\n\nA central do Neon Flow criou o seu acesso de parceiro. Para definir a sua senha, abra o link abaixo em até 7 dias:\n\n%s/conta/convite/%s\n\n"+
			"Depois, entre com este e-mail: %s\n\nNeon Flow", p.Name, s.appURL, token, p.Email),
	})
	if err != nil {
		return account.Invite{}, err
	}
	return account.Invite{SentTo: p.Email, ExpiresAt: expiresAt}, nil
}

func (s *AccountService) AcceptInvite(ctx context.Context, req account.PasswordRequest) error {
	hash, err := s.passwordFromLink(req)
	if err != nil {
		return err
	}
	return s.repo.AcceptInvite(ctx, account.HashToken(req.Token), hash)
}

func (s *AccountService) passwordFromLink(req account.PasswordRequest) (string, error) {
	if req.Token == "" {
		return "", account.ErrInvalidToken
	}
	if len(req.Password) < authentication.MinPasswordLength {
		return "", authentication.ErrWeakPassword
	}
	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}
	return hash, nil
}

// issue grava o hash de um código novo e devolve o código, que só vai no e-mail.
func (s *AccountService) issue(ctx context.Context, t account.Token, ttl time.Duration) (string, error) {
	token, hash, err := account.NewToken()
	if err != nil {
		return "", err
	}
	t.Hash = hash
	t.ExpiresAt = s.now().Add(ttl)
	if err := s.repo.SaveToken(ctx, t); err != nil {
		return "", err
	}
	return token, nil
}

// send registra a falha do provedor no log, sem o destinatário, e devolve uma mensagem para a pessoa.
func (s *AccountService) send(ctx context.Context, m email.Message) error {
	if err := s.mail.Send(ctx, m); err != nil {
		s.log.Error("email send failed", "subject", m.Subject, "error", err)
		return account.ErrSendFailed
	}
	return nil
}
