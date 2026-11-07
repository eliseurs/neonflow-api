// Package maintenance é o histórico de manutenção do veículo, no painel: base para comparar quanto
// cada carro custa e quando parou (pedido do fundador em 06/11/2026). Custo é informativo; quem paga
// a manutenção do carro parceiro é o dono (RN-03.5).
package maintenance

import (
	"strings"
	"time"
	"unicode/utf8"

	"neonflow-api/internal/pkg/timezone"
)

const (
	DateLayout     = "2006-01-02"
	maxDescription = 1000
)

// Kinds são os tipos de manutenção.
var Kinds = []string{"preventiva", "corretiva", "revisao", "pneus", "outros"}

type Record struct {
	ID          string    `json:"id"`
	VehicleID   string    `json:"vehicle_id"`
	Date        string    `json:"date"`
	Kind        string    `json:"kind"`
	Description string    `json:"description"`
	CostCents   *int64    `json:"cost_cents"`
	OdometerKm  *int      `json:"odometer_km"`
	Workshop    string    `json:"workshop"`
	CreatedAt   time.Time `json:"created_at"`
}

type Request struct {
	Date        string `json:"date"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	CostCents   *int64 `json:"cost_cents"`
	OdometerKm  *int   `json:"odometer_km"`
	Workshop    string `json:"workshop"`
}

// Validate: data válida e não futura (no dia de Gramado), tipo conhecido, descrição, custo e km não negativos.
func (r Request) Validate(now time.Time) (Request, error) {
	r.Date = strings.TrimSpace(r.Date)
	r.Kind = strings.ToLower(strings.TrimSpace(r.Kind))
	r.Description = strings.TrimSpace(r.Description)
	r.Workshop = strings.TrimSpace(r.Workshop)
	d, err := time.ParseInLocation(DateLayout, r.Date, timezone.Location)
	switch {
	case err != nil:
		return r, ErrInvalidDate
	case d.After(now.In(timezone.Location)):
		return r, ErrFutureDate
	case !known(r.Kind):
		return r, ErrInvalidKind
	case r.Description == "":
		return r, ErrDescriptionRequired
	case utf8.RuneCountInString(r.Description) > maxDescription:
		return r, ErrDescriptionTooLong
	case r.CostCents != nil && *r.CostCents < 0:
		return r, ErrInvalidCost
	case r.OdometerKm != nil && *r.OdometerKm < 0:
		return r, ErrInvalidOdometer
	}
	return r, nil
}

func known(kind string) bool {
	for _, k := range Kinds {
		if k == kind {
			return true
		}
	}
	return false
}
