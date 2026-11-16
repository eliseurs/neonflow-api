// Package audit é a consulta da trilha de auditoria das rotas (RN-05.9). A gravação fica no middleware Audit.
package audit

import "time"

const (
	DefaultLimit = 100
	MaxLimit     = 500
)

type Entry struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	UserName string    `json:"user_name"`
	Role     string    `json:"role"`
	Method   string    `json:"method"`
	Route    string    `json:"route"`
	Path     string    `json:"path"`
	Status   int       `json:"status"`
	At       time.Time `json:"created_at"`
}

// Filter da listagem; campos vazios não filtram. Denied traz só os acessos negados (403).
type Filter struct {
	UserID string
	Denied bool
	Limit  int
}

// Normalize aplica o limite padrão e o máximo.
func (f Filter) Normalize() (Filter, error) {
	switch {
	case f.Limit < 0 || f.Limit > MaxLimit:
		return f, ErrInvalidLimit
	case f.Limit == 0:
		f.Limit = DefaultLimit
	}
	return f, nil
}
