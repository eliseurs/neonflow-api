// Package account cria e recupera as contas de parceiros e clientes (RN-05.11 a RN-05.14): o cliente se
// cadastra no site e confirma o e-mail, o parceiro recebe o convite da central, e qualquer conta pede
// uma senha nova por e-mail. Login e sessão continuam no domínio authentication.
package account

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"neonflow-api/internal/domain/authentication"
)

// Finalidade de cada link enviado por e-mail.
const (
	PurposeConfirmEmail  = "verificar_email"
	PurposeInvite        = "convite"
	PurposeResetPassword = "nova_senha"
)

// Validade dos links. Decisão técnica (RN-05.14): o convite espera o parceiro por uma semana, a
// confirmação um dia, e a nova senha uma hora, porque dá acesso à conta.
const (
	ConfirmEmailTTL  = 24 * time.Hour
	InviteTTL        = 7 * 24 * time.Hour
	ResetPasswordTTL = time.Hour
)

// SignupRequest é o cadastro do cliente no site (RN-05.13).
type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// EmailRequest é o pedido de nova senha.
type EmailRequest struct {
	Email string `json:"email"`
}

// TokenRequest é a confirmação do e-mail, com o código do link.
type TokenRequest struct {
	Token string `json:"token"`
}

// PasswordRequest define a senha a partir de um link: nova senha ou convite aceito.
type PasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// Account é a conta encontrada pelo e-mail, sem o hash da senha.
type Account struct {
	ID       string
	Name     string
	Email    string
	Role     string
	Verified bool
}

// NewCustomerAccount é o que o repository grava no cadastro: dados validados e senha com hash.
type NewCustomerAccount struct {
	Name         string
	Email        string
	PasswordHash string
}

// Token é o link gravado: só o hash do código, nunca o código.
type Token struct {
	Purpose   string
	Hash      string
	UserID    string
	PartnerID string
	ExpiresAt time.Time
}

// PartnerContact é o parceiro a convidar e o que impede o convite.
type PartnerContact struct {
	Name       string
	Email      string
	Active     bool
	HasAccount bool
	// EmailTaken: outra conta (da equipe ou de cliente) já usa este e-mail.
	EmailTaken bool
}

// Invite é a resposta à central: para onde foi o convite e até quando vale.
type Invite struct {
	SentTo    string    `json:"sent_to"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Validate normaliza e valida o cadastro: nome, e-mail e senha de 10+ caracteres, como no painel.
func (r SignupRequest) Validate() (SignupRequest, error) {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = authentication.NormalizeEmail(r.Email)
	switch {
	case r.Name == "":
		return r, authentication.ErrNameRequired
	case !ValidEmail(r.Email):
		return r, authentication.ErrInvalidEmail
	case len(r.Password) < authentication.MinPasswordLength:
		return r, authentication.ErrWeakPassword
	}
	return r, nil
}

// ValidEmail aceita só o endereço puro, sem nome ("Ana <ana@x>").
func ValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

// NewToken sorteia o código do link (32 bytes) e devolve também o hash que vai para o banco.
func NewToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}
