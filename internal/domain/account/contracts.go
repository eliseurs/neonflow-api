package account

import (
	"context"

	"neonflow-api/internal/infra/email"
)

type Repository interface {
	// CreateCustomerAccount grava a conta do cliente ainda sem o e-mail confirmado; e-mail repetido é ErrEmailTaken.
	CreateCustomerAccount(ctx context.Context, n NewCustomerAccount) (Account, error)
	// FindAccount devolve a conta ativa com o e-mail; ErrUserNotFound se não houver.
	FindAccount(ctx context.Context, email string) (Account, error)
	SaveToken(ctx context.Context, t Token) error
	// ConfirmEmail usa o link, confirma o e-mail, liga a conta ao cadastro do cliente e envia à central os
	// pedidos que aguardavam a confirmação.
	ConfirmEmail(ctx context.Context, tokenHash string) (Confirmed, error)
	ResetPassword(ctx context.Context, tokenHash, passwordHash string) error
	PartnerForInvite(ctx context.Context, partnerID string) (PartnerContact, error)
	// AcceptInvite usa o convite e cria a conta do parceiro com a senha escolhida.
	AcceptInvite(ctx context.Context, tokenHash, passwordHash string) error
}

type Service interface {
	Signup(ctx context.Context, req SignupRequest) error
	ConfirmEmail(ctx context.Context, req TokenRequest) (Confirmation, error)
	ForgotPassword(ctx context.Context, req EmailRequest) error
	ResetPassword(ctx context.Context, req PasswordRequest) error
	InvitePartner(ctx context.Context, partnerID string) (Invite, error)
	AcceptInvite(ctx context.Context, req PasswordRequest) error
}

type Mailer interface {
	Enabled() bool
	Send(ctx context.Context, m email.Message) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
