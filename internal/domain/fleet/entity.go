package fleet

import (
	"regexp"
	"strings"
	"time"
)

// Categorias aprovadas em 03/11/2026 (RN-01.4).
const (
	CategoryEconomy = "economy"
	CategoryPremium = "premium"

	GearManual     = "manual"
	GearAutomatico = "automatico"
)

// Estados exibidos do veículo, em ordem de prioridade (RN-01.1, RN-01.3, RN-01.6).
const (
	StatusManutencao          = "manutencao"
	StatusEmLocacao           = "em_locacao"
	StatusAguardandoChecklist = "aguardando_checklist"
	StatusReservado           = "reservado"
	StatusLivre               = "livre"
)

// Placa antiga (ABC1234) ou Mercosul (ABC1D23), já sem hífen e em maiúsculas.
var platePattern = regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z0-9][0-9]{2}$`)

type Vehicle struct {
	ID             string    `json:"id"`
	Plate          string    `json:"plate"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	Owner          string    `json:"owner"`
	Seats          int       `json:"seats"`
	Gear           string    `json:"gear"`
	DailyRateCents int64     `json:"daily_rate_cents"`
	InMaintenance  bool      `json:"in_maintenance"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	// Status e NextBooking vêm calculados nas leituras da frota; outras listas (disponibilidade) não os trazem.
	Status      string       `json:"status,omitempty"`
	NextBooking *NextBooking `json:"next_booking,omitempty"`
}

// NextBooking é a próxima reserva ativa que ainda não terminou: confirmada ou pré-reserva no prazo (RN-01.3).
type NextBooking struct {
	ID         string    `json:"id"`
	ClientName string    `json:"client_name"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	StartDate  string    `json:"start_date"` // dia de StartAt em Gramado; mantido por compatibilidade
	EndDate    string    `json:"end_date"`
	Stage      string    `json:"stage"`
}

// VehicleRequest é o corpo de criação e de atualização (PUT substitui todos os campos).
type VehicleRequest struct {
	Plate          string `json:"plate"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Owner          string `json:"owner"`
	Seats          int    `json:"seats"`
	Gear           string `json:"gear"`
	DailyRateCents int64  `json:"daily_rate_cents"`
	InMaintenance  bool   `json:"in_maintenance"`
}

func NormalizePlate(plate string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(plate), "-", ""))
}

// Validate normaliza e valida o veículo. A diária é dado da central, nunca default (RN-01.5).
func (r VehicleRequest) Validate() (VehicleRequest, error) {
	r.Plate = NormalizePlate(r.Plate)
	r.Name = strings.TrimSpace(r.Name)
	r.Category = strings.ToLower(strings.TrimSpace(r.Category))
	r.Owner = strings.TrimSpace(r.Owner)
	r.Gear = strings.ToLower(strings.TrimSpace(r.Gear))

	switch {
	case !platePattern.MatchString(r.Plate):
		return r, ErrInvalidPlate
	case r.Name == "":
		return r, ErrNameRequired
	case r.Category != CategoryEconomy && r.Category != CategoryPremium:
		return r, ErrInvalidCategory
	case r.Owner == "":
		return r, ErrOwnerRequired
	case r.Seats <= 0:
		return r, ErrInvalidSeats
	case r.Gear != GearManual && r.Gear != GearAutomatico:
		return r, ErrInvalidGear
	case r.DailyRateCents <= 0:
		return r, ErrInvalidDailyRate
	}
	return r, nil
}
