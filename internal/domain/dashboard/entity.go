// Package dashboard são os indicadores da tela inicial da central (PEN-11, q79, q97, q98).
package dashboard

import (
	"regexp"
	"time"

	"neonflow-api/internal/pkg/timezone"
)

var monthPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// Summary é o resumo do mês e de hoje.
type Summary struct {
	Month string `json:"month"`
	// Reservas com retirada no mês, confirmadas em diante: diárias + atraso + km (q97).
	PickupRevenueCents int64 `json:"pickup_revenue_cents"`
	PickupCount        int   `json:"pickup_count"`
	// Reservas cujo pagamento foi marcado no mês, pelo valor das diárias (q97).
	PaidCents int64 `json:"paid_cents"`
	PaidCount int   `json:"paid_count"`
	// Ocupação = carros em locação agora ÷ frota cadastrada (q98).
	RentedToday      int     `json:"rented_today"`
	FleetSize        int     `json:"fleet_size"`
	OccupancyPercent float64 `json:"occupancy_percent"`
}

// Totals é o que o repository soma no período [Start, End).
type Totals struct {
	PickupRevenueCents int64
	PickupCount        int
	PaidCents          int64
	PaidCount          int
	RentedToday        int
	FleetSize          int
}

// MonthRange devolve o início do mês (AAAA-MM) e do mês seguinte no fuso de Gramado. Vazio = mês atual.
func MonthRange(month string, now time.Time) (string, time.Time, time.Time, error) {
	if month == "" {
		month = now.In(timezone.Location).Format("2006-01")
	}
	if !monthPattern.MatchString(month) {
		return "", time.Time{}, time.Time{}, ErrInvalidMonth
	}
	start, err := time.ParseInLocation("2006-01", month, timezone.Location)
	if err != nil {
		return "", time.Time{}, time.Time{}, ErrInvalidMonth
	}
	return month, start, start.AddDate(0, 1, 0), nil
}

// Occupancy em percentual com uma casa; frota vazia = 0.
func Occupancy(rented, fleet int) float64 {
	if fleet == 0 {
		return 0
	}
	return float64(rented*1000/fleet) / 10
}
