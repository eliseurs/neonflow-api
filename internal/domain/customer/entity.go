// Package customer é o cadastro de clientes, fase 1: contato, idioma e histórico de pedidos (RN-04.7).
// Documentos esperam a política de retenção (PEN-48). Dados pessoais nunca vão para o log.
package customer

import (
	"net/mail"
	"regexp"
	"strings"
	"time"
)

const (
	LangPT = "pt"
	LangEN = "en"

	// AnonymizedName substitui o nome de quem pediu a eliminação dos dados.
	AnonymizedName = "Cliente anonimizado"
)

var nonDigits = regexp.MustCompile(`\D`)

type Customer struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	WhatsApp     string     `json:"whatsapp"`
	Email        string     `json:"email"`
	Language     string     `json:"language"`
	Notes        string     `json:"notes"`
	AnonymizedAt *time.Time `json:"anonymized_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	// Contagem de pedidos, na listagem.
	BookingCount  int `json:"booking_count"`
	TransferCount int `json:"transfer_count"`
}

// Order é um pedido do cliente no histórico: reserva de aluguel ou transfer.
type Order struct {
	Kind       string    `json:"kind"` // "reserva" ou "transfer"
	ID         string    `json:"id"`
	At         time.Time `json:"at"`
	Summary    string    `json:"summary"`
	Stage      string    `json:"stage"`
	TotalCents int64     `json:"total_cents"`
}

// Detail é o cliente com o histórico, mais recentes primeiro.
type Detail struct {
	Customer
	Orders []Order `json:"orders"`
}

type Request struct {
	Name     string `json:"name"`
	WhatsApp string `json:"whatsapp"`
	Email    string `json:"email"`
	Language string `json:"language"`
	Notes    string `json:"notes"`
}

func (r Request) Validate() (Request, error) {
	r.Name = strings.TrimSpace(r.Name)
	r.WhatsApp = nonDigits.ReplaceAllString(r.WhatsApp, "")
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Language = strings.ToLower(strings.TrimSpace(r.Language))
	r.Notes = strings.TrimSpace(r.Notes)
	if r.Language == "" {
		r.Language = LangPT
	}
	switch {
	case r.Name == "":
		return r, ErrNameRequired
	case r.WhatsApp != "" && (len(r.WhatsApp) < 10 || len(r.WhatsApp) > 15):
		return r, ErrInvalidWhatsApp
	case r.Email != "" && !validEmail(r.Email):
		return r, ErrInvalidEmail
	case r.Language != LangPT && r.Language != LangEN:
		return r, ErrInvalidLanguage
	}
	return r, nil
}

// NormalizeQuery prepara a busca: texto em minúsculas e, se houver, só os dígitos (para WhatsApp).
func NormalizeQuery(q string) (text, digits string) {
	text = strings.ToLower(strings.TrimSpace(q))
	digits = nonDigits.ReplaceAllString(text, "")
	return text, digits
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}
