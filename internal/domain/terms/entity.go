// Package terms é o aceite dos termos do aluguel pelo cliente (RN-02.3, q38). O texto é do jurídico:
// o admin publica em versões; a central envia o link da reserva; o cliente aceita digitando o nome.
// Quando o contrato 04 (Ordem de Locação, RN-08) tem versão publicada, o link mostra a Ordem e o
// quadro da reserva, e o aceite segue o padrão V15: versão exata, hashes, confirmação e método.
package terms

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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
	// OrderCode é o contrato da reserva sem motorista (RN-08.1).
	OrderCode = "04"
	// MethodLink: o cliente abre o link secreto enviado pela central e digita o nome.
	MethodLink = "link_reserva"
	DocTerms   = "termos"
	DocOrder   = "ordem_04"
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

// Acceptance: Document diz o que foi aceito (termos antigos ou Ordem 04); SHA256 só na Ordem.
type Acceptance struct {
	Document   string    `json:"document"`
	Version    int       `json:"version"`
	SHA256     string    `json:"sha256,omitempty"`
	TypedName  string    `json:"typed_name"`
	AcceptedAt time.Time `json:"accepted_at"`
}

// Order é a versão publicada da Ordem 04 que o cliente lê.
type Order struct {
	VersionID string `json:"version_id"`
	Version   int    `json:"version"`
	Label     string `json:"label"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	SHA256    string `json:"sha256"`
}

// Quadro são os dados da reserva mostrados ao cliente com a Ordem; o hash dele vai no aceite.
// Caução e km são as cópias gravadas na reserva (nulo = não informado ou km livre).
type Quadro struct {
	ClientName    string    `json:"client_name"`
	VehicleName   string    `json:"vehicle_name"`
	Place         string    `json:"place"`
	StartAt       time.Time `json:"start_at"`
	EndAt         time.Time `json:"end_at"`
	TotalCents    int64     `json:"total_cents"`
	DepositCents  *int64    `json:"deposit_cents"`
	KmLimitPerDay *int      `json:"km_limit_per_day"`
	ExtraKmCents  *int64    `json:"extra_km_cents"`
}

// Hash do quadro como JSON (mesma ordem de campos sempre).
func (q Quadro) Hash() (string, []byte, error) {
	b, err := json.Marshal(q)
	if err != nil {
		return "", nil, fmt.Errorf("quadro: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), b, nil
}

// Link é o que a central recebe para enviar ao cliente; o código só aparece aqui.
type Link struct {
	Token     string    `json:"token"`
	Path      string    `json:"path"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Public é o que a página do cliente mostra: o quadro da reserva e a Ordem 04 publicada ou, sem ela,
// os termos atuais (um dos dois, nunca os dois).
type Public struct {
	Quadro
	QuadroSHA256 string      `json:"quadro_sha256"`
	Order        *Order      `json:"order"`
	Terms        *Version    `json:"terms"`
	Accepted     *Acceptance `json:"accepted"`
	ExpiresAt    time.Time   `json:"expires_at"`
}

// LinkInfo é o link encontrado pelo hash, com a reserva.
type LinkInfo struct {
	ID           string
	BookingID    string
	BookingStage string
	ExpiresAt    time.Time
}

// AcceptRequest: termos antigos mandam `version`; a Ordem 04 manda `contract_version_id`, o `sha256`
// do texto, o `quadro_sha256` lido e `confirm` (caixa marcada pelo cliente).
type AcceptRequest struct {
	TypedName         string `json:"typed_name"`
	Version           int    `json:"version"`
	ContractVersionID string `json:"contract_version_id"`
	SHA256            string `json:"sha256"`
	QuadroSHA256      string `json:"quadro_sha256"`
	Confirm           bool   `json:"confirm"`
}

// IsOrder: o aceite é da Ordem 04.
func (r AcceptRequest) IsOrder() bool { return r.ContractVersionID != "" }

func (r AcceptRequest) Validate() (AcceptRequest, error) {
	r.TypedName = strings.Join(strings.Fields(r.TypedName), " ")
	r.ContractVersionID = strings.TrimSpace(r.ContractVersionID)
	r.SHA256 = strings.ToLower(strings.TrimSpace(r.SHA256))
	r.QuadroSHA256 = strings.ToLower(strings.TrimSpace(r.QuadroSHA256))
	n := utf8.RuneCountInString(r.TypedName)
	switch {
	case n < minName || n > maxName:
		return r, ErrInvalidName
	case r.IsOrder() && !r.Confirm:
		return r, ErrConfirmRequired
	case r.IsOrder() && (r.SHA256 == "" || r.QuadroSHA256 == ""):
		return r, ErrVersionChanged
	case !r.IsOrder() && r.Version <= 0:
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
