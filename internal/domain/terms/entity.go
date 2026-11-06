// Package terms é o aceite dos termos do aluguel pelo cliente (RN-02.3, q38). O texto é do jurídico:
// o admin publica em versões; a central envia o link da reserva; o cliente aceita digitando o nome.
package terms

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// LinkTTL: o link de aceite vale 7 dias.
	LinkTTL = 7 * 24 * time.Hour
	// TermsCheck é a conferência da reserva marcada pelo aceite (RN-02.3).
	TermsCheck = "terms"
	minName    = 3
	maxName    = 200
	maxBody    = 100_000
)

type Version struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type Acceptance struct {
	Version    int       `json:"version"`
	TypedName  string    `json:"typed_name"`
	AcceptedAt time.Time `json:"accepted_at"`
}

// Link é o que a central recebe para enviar ao cliente; o código só aparece aqui.
type Link struct {
	Token     string    `json:"token"`
	Path      string    `json:"path"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Public é o que a página do cliente mostra: o resumo da reserva e os termos atuais.
type Public struct {
	ClientName  string      `json:"client_name"`
	VehicleName string      `json:"vehicle_name"`
	StartAt     time.Time   `json:"start_at"`
	EndAt       time.Time   `json:"end_at"`
	TotalCents  int64       `json:"total_cents"`
	Terms       Version     `json:"terms"`
	Accepted    *Acceptance `json:"accepted"`
	ExpiresAt   time.Time   `json:"expires_at"`
}

// LinkInfo é o link encontrado pelo hash, com a reserva.
type LinkInfo struct {
	ID           string
	BookingID    string
	BookingStage string
	ExpiresAt    time.Time
}

type AcceptRequest struct {
	TypedName string `json:"typed_name"`
	Version   int    `json:"version"`
}

func (r AcceptRequest) Validate() (AcceptRequest, error) {
	r.TypedName = strings.Join(strings.Fields(r.TypedName), " ")
	n := utf8.RuneCountInString(r.TypedName)
	switch {
	case n < minName || n > maxName:
		return r, ErrInvalidName
	case r.Version <= 0:
		return r, ErrVersionChanged
	}
	return r, nil
}

// ValidateBody: texto não vazio e de tamanho razoável.
func ValidateBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", ErrBodyRequired
	}
	if len(body) > maxBody {
		return "", ErrBodyTooLong
	}
	return body, nil
}

// LinkAllowed: só antes da retirada (pré-reserva ou confirmada).
func LinkAllowed(bookingStage string) bool {
	return bookingStage == "preparacao" || bookingStage == "confirmada"
}

// NewToken gera o código do link (32 bytes aleatórios) e o hash guardado no banco.
func NewToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
