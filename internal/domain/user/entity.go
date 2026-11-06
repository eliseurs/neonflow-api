// Package user é a gestão de usuários do painel pelo admin (RN-05.2, RN-05.6).
// Login e sessão continuam no domínio authentication.
package user

import (
	"strings"
	"time"

	"neonflow-api/internal/domain/authentication"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UpdateRequest troca nome, papel e ativo. O e-mail não muda (é o login).
type UpdateRequest struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Active *bool  `json:"active"`
}

// PasswordRequest é a nova senha definida pelo admin para outro usuário.
type PasswordRequest struct {
	Password string `json:"password"`
}

// ChangeOwnPasswordRequest é a troca da própria senha, com a senha atual.
type ChangeOwnPasswordRequest struct {
	Current  string `json:"current_password"`
	Password string `json:"new_password"`
}

func (r UpdateRequest) Validate() (UpdateRequest, error) {
	r.Name = strings.TrimSpace(r.Name)
	r.Role = strings.TrimSpace(r.Role)
	switch {
	case r.Name == "":
		return r, authentication.ErrNameRequired
	case !authentication.ValidRole(r.Role):
		return r, authentication.ErrInvalidRole
	}
	return r, nil
}

func ValidPassword(p string) error {
	if len(p) < authentication.MinPasswordLength {
		return authentication.ErrWeakPassword
	}
	return nil
}
