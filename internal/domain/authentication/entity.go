package authentication

import (
	"net/mail"
	"strings"
	"time"

	"neonflow-api/internal/pkg/authz"
)

const (
	RoleAdmin   = authz.RoleAdmin
	RoleCentral = authz.RoleCentral
	// RoleLegalReviewer é a revisora jurídica dos contratos (06_GUIA_AREA_JURIDICA, pacote V15). Não acessa a operação.
	RoleLegalReviewer = authz.RoleLegalReviewer

	MinPasswordLength = 10
)

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
	// Permissions é o que o papel pode abrir (RN-05.10); o app monta o menu e as telas por esta lista.
	Permissions []string `json:"permissions"`
	// PartnerID e CustomerID ligam a conta ao cadastro do parceiro ou do cliente (RN-05.11).
	PartnerID  *string   `json:"partner_id,omitempty"`
	CustomerID *string   `json:"customer_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Credentials é o usuário com o hash da senha. Nunca sai da API.
type Credentials struct {
	User
	PasswordHash  string
	EmailVerified bool
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

type CreateUserRequest struct {
	Name     string
	Email    string
	Password string
	Role     string
}

// NewUser é o que o repository grava: dados já validados e senha já com hash.
type NewUser struct {
	Name         string
	Email        string
	PasswordHash string
	Role         string
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidRole diz se o papel pode ser criado pela CLI ou pelo admin: só os da equipe (RN-05.11).
func ValidRole(role string) bool {
	return authz.StaffRole(role)
}

// Validate normaliza e valida o pedido de criação de usuário.
func (r CreateUserRequest) Validate() (CreateUserRequest, error) {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = NormalizeEmail(r.Email)
	r.Role = strings.TrimSpace(r.Role)

	switch {
	case r.Name == "":
		return r, ErrNameRequired
	case !validEmail(r.Email):
		return r, ErrInvalidEmail
	case len(r.Password) < MinPasswordLength:
		return r, ErrWeakPassword
	case !ValidRole(r.Role):
		return r, ErrInvalidRole
	}
	return r, nil
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}
