// Package partner é o cadastro de parceiros (RN-07): proprietários de veículo, motoristas e
// indicadores. Qualquer parceiro pode indicar, pelo link ou QR com o seu código.
package partner

import (
	"crypto/rand"
	"math"
	"math/big"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

const (
	KindOwner    = "proprietario"
	KindDriver   = "motorista"
	KindReferrer = "indicador"

	// ReferralCodeLength e o alfabeto sem caracteres ambíguos (0/O, 1/I/L) para ler e digitar.
	ReferralCodeLength = 6
	referralAlphabet   = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

var (
	nonDigits      = regexp.MustCompile(`\D`)
	referralFormat = regexp.MustCompile(`^[A-Z0-9]{6,12}$`)
)

type Partner struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Kinds        []string `json:"kinds"`
	WhatsApp     string   `json:"whatsapp"`
	Email        string   `json:"email"`
	City         string   `json:"city"`
	Notes        string   `json:"notes"`
	ReferralCode string   `json:"referral_code"`
	// Percentuais configuráveis por parceiro; nulo = não definido (RN-03.4, 03.7, 03.9).
	CommissionPercent  *float64  `json:"commission_percent"`
	OwnerSharePercent  *float64  `json:"owner_share_percent"`
	DriverSharePercent *float64  `json:"driver_share_percent"`
	Active             bool      `json:"active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Request é o corpo de criação e de atualização. O código de indicação é gerado e não muda.
type Request struct {
	Name               string   `json:"name"`
	Kinds              []string `json:"kinds"`
	WhatsApp           string   `json:"whatsapp"`
	Email              string   `json:"email"`
	City               string   `json:"city"`
	Notes              string   `json:"notes"`
	CommissionPercent  *float64 `json:"commission_percent"`
	OwnerSharePercent  *float64 `json:"owner_share_percent"`
	DriverSharePercent *float64 `json:"driver_share_percent"`
	Active             *bool    `json:"active"`
}

func (r Request) Has(kind string) bool {
	for _, k := range r.Kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// Validate normaliza e valida. Repasse ao proprietário só vale para proprietário, e o percentual
// do motorista só para motorista; fora disso, são descartados.
func (r Request) Validate() (Request, error) {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.City = strings.TrimSpace(r.City)
	r.Notes = strings.TrimSpace(r.Notes)
	r.WhatsApp = nonDigits.ReplaceAllString(r.WhatsApp, "")

	kinds := []string{}
	seen := map[string]bool{}
	for _, k := range r.Kinds {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != KindOwner && k != KindDriver && k != KindReferrer {
			return r, ErrInvalidKind
		}
		if !seen[k] {
			seen[k] = true
			kinds = append(kinds, k)
		}
	}
	r.Kinds = kinds

	switch {
	case r.Name == "":
		return r, ErrNameRequired
	case len(r.Kinds) == 0:
		return r, ErrInvalidKind
	case r.Email != "" && !validEmail(r.Email):
		return r, ErrInvalidEmail
	}
	for _, p := range []*float64{r.CommissionPercent, r.OwnerSharePercent, r.DriverSharePercent} {
		if p != nil && (*p < 0 || *p > 100 || math.IsNaN(*p)) {
			return r, ErrInvalidPercent
		}
	}
	if !r.Has(KindOwner) {
		r.OwnerSharePercent = nil
	}
	if !r.Has(KindDriver) {
		r.DriverSharePercent = nil
	}
	return r, nil
}

// ToBasisPoints converte percentual (12,5) em pontos-base (1250).
func ToBasisPoints(p *float64) *int {
	if p == nil {
		return nil
	}
	bp := int(math.Round(*p * 100))
	return &bp
}

// FromBasisPoints faz o caminho inverso.
func FromBasisPoints(bp *int) *float64 {
	if bp == nil {
		return nil
	}
	p := float64(*bp) / 100
	return &p
}

// NewReferralCode gera um código aleatório de 6 caracteres.
func NewReferralCode() (string, error) {
	var b strings.Builder
	max := big.NewInt(int64(len(referralAlphabet)))
	for i := 0; i < ReferralCodeLength; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b.WriteByte(referralAlphabet[n.Int64()])
	}
	return b.String(), nil
}

// NormalizeReferralCode limpa um código digitado ou vindo de link; vazio se não for válido.
func NormalizeReferralCode(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	if !referralFormat.MatchString(c) {
		return ""
	}
	return c
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}
