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

// Combustíveis aceitos na ficha; vazio = não informado.
var Fuels = []string{"flex", "gasolina", "etanol", "diesel", "eletrico", "hibrido"}

// Features são os itens marcáveis na ficha do carro (rótulos no painel e no site).
var Features = []string{
	"ar_condicionado", "direcao_eletrica", "vidros_eletricos", "multimidia", "bluetooth",
	"camera_re", "sensor_estacionamento", "piloto_automatico", "bancos_couro", "teto_solar",
}

const maxDescription = 2000

// Placa antiga (ABC1234) ou Mercosul (ABC1D23), já sem hífen e em maiúsculas.
var platePattern = regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z0-9][0-9]{2}$`)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Vehicle struct {
	ID             string `json:"id"`
	Plate          string `json:"plate"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Owner          string `json:"owner"`
	Seats          int    `json:"seats"`
	Gear           string `json:"gear"`
	DailyRateCents int64  `json:"daily_rate_cents"`
	InMaintenance  bool   `json:"in_maintenance"`
	// Caução (RN-06.4) e quilometragem (RN-02.18), configuráveis; nulo = não informado.
	// Parceiro proprietário cadastrado (RN-07.4); Owner continua como nome livre.
	OwnerPartnerID *string `json:"owner_partner_id"`
	DepositCents   *int64  `json:"deposit_cents"`
	KmLimitPerDay  *int    `json:"km_limit_per_day"`
	ExtraKmCents   *int64  `json:"extra_km_cents"`
	// Ficha para o site e o painel; todos opcionais.
	Year        *int      `json:"year"`
	Color       string    `json:"color"`
	Fuel        string    `json:"fuel"`
	Doors       *int      `json:"doors"`
	Features    []string  `json:"features"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Status e NextBooking vêm calculados nas leituras da frota; outras listas (disponibilidade) não os trazem.
	Status      string       `json:"status,omitempty"`
	NextBooking *NextBooking `json:"next_booking,omitempty"`
}

// StatsWindow é o período do desempenho: os últimos 12 meses.
const StatsWindow = 365 * 24 * time.Hour

// Stats é o desempenho do carro no período, para comparar a frota: quanto sai, quanto rende e
// quanto custa em manutenção. Conta as reservas confirmadas em diante com retirada no período.
type Stats struct {
	VehicleID        string  `json:"vehicle_id"`
	Name             string  `json:"name"`
	Plate            string  `json:"plate"`
	Bookings         int     `json:"bookings"`
	RentedDays       int     `json:"rented_days"`
	OccupancyPercent float64 `json:"occupancy_percent"`
	RevenueCents     int64   `json:"revenue_cents"`
	KmDriven         int     `json:"km_driven"`
	MaintenanceCount int     `json:"maintenance_count"`
	MaintenanceCents int64   `json:"maintenance_cents"`
	// ResultCents = receita - manutenção (sem repasse, comissão e outros custos).
	ResultCents     int64     `json:"result_cents"`
	LastMaintenance *string   `json:"last_maintenance"`
	Since           time.Time `json:"since"`
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
	Plate          string   `json:"plate"`
	Name           string   `json:"name"`
	Category       string   `json:"category"`
	Owner          string   `json:"owner"`
	Seats          int      `json:"seats"`
	Gear           string   `json:"gear"`
	DailyRateCents int64    `json:"daily_rate_cents"`
	InMaintenance  bool     `json:"in_maintenance"`
	DepositCents   *int64   `json:"deposit_cents"`
	KmLimitPerDay  *int     `json:"km_limit_per_day"`
	ExtraKmCents   *int64   `json:"extra_km_cents"`
	OwnerPartnerID *string  `json:"owner_partner_id"`
	Year           *int     `json:"year"`
	Color          string   `json:"color"`
	Fuel           string   `json:"fuel"`
	Doors          *int     `json:"doors"`
	Features       []string `json:"features"`
	Description    string   `json:"description"`
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
	r.Color = strings.TrimSpace(r.Color)
	r.Fuel = strings.ToLower(strings.TrimSpace(r.Fuel))
	r.Description = strings.TrimSpace(r.Description)
	features, ok := normalizeFeatures(r.Features)
	if !ok {
		return r, ErrInvalidFeature
	}
	r.Features = features

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
	case r.OwnerPartnerID != nil && !uuidPattern.MatchString(*r.OwnerPartnerID):
		return r, ErrOwnerPartnerNotFound
	case r.DepositCents != nil && *r.DepositCents <= 0:
		return r, ErrInvalidDeposit
	case r.KmLimitPerDay != nil && *r.KmLimitPerDay <= 0, r.ExtraKmCents != nil && *r.ExtraKmCents <= 0:
		return r, ErrInvalidKmPlan
	case r.ExtraKmCents != nil && r.KmLimitPerDay == nil:
		return r, ErrKmPlanIncomplete
	case r.DailyRateCents <= 0:
		return r, ErrInvalidDailyRate
	case r.Year != nil && (*r.Year < 1980 || *r.Year > 2100):
		return r, ErrInvalidYear
	case r.Fuel != "" && !contains(Fuels, r.Fuel):
		return r, ErrInvalidFuel
	case r.Doors != nil && (*r.Doors < 2 || *r.Doors > 5):
		return r, ErrInvalidDoors
	case len([]rune(r.Description)) > maxDescription:
		return r, ErrDescriptionTooLong
	}
	return r, nil
}

// normalizeFeatures tira repetidos e mantém a ordem da lista oficial; item desconhecido é recusado.
func normalizeFeatures(in []string) ([]string, bool) {
	picked := map[string]bool{}
	for _, f := range in {
		f = strings.ToLower(strings.TrimSpace(f))
		if !contains(Features, f) {
			return nil, false
		}
		picked[f] = true
	}
	out := []string{}
	for _, f := range Features {
		if picked[f] {
			out = append(out, f)
		}
	}
	return out, true
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
