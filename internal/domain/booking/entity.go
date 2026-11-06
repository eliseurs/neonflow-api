package booking

import (
	"fmt"
	"strings"
	"time"
)

// Etapas da locação (RN-02.4). Ativas são as que bloqueiam o período: todas, menos encerrada e cancelada.
const (
	StagePreparacao = "preparacao"
	StageConfirmada = "confirmada"
	StageEmLocacao  = "em_locacao"
	StageDevolvida  = "devolvida"
	StageEncerrada  = "encerrada"
	StageCancelada  = "cancelada"

	// DateLayout é o formato das datas de retirada e devolução (RN-02.7).
	DateLayout = "2006-01-02"

	// PaymentWindow é o prazo para pagar a pré-reserva; depois ela é cancelada (RN-02.8, RN-02.11, ADR-0012).
	PaymentWindow = 24 * time.Hour
)

// Conferências exigidas para sair de cada etapa (RN-02.3, RN-02.4). "payment" é o pagamento
// integral confirmado pela central (RN-02.14); "terms" é o aceite dos termos (ADR-0013).
var requiredChecks = map[string][]string{
	StagePreparacao: {"cnh", "terms", "deposit", "payment"},
	StageConfirmada: {"delivery"},
	StageEmLocacao:  {"return"},
	StageDevolvida:  {"issues"},
}

var nextStage = map[string]string{
	StagePreparacao: StageConfirmada,
	StageConfirmada: StageEmLocacao,
	StageEmLocacao:  StageDevolvida,
	StageDevolvida:  StageEncerrada,
}

type Booking struct {
	ID             string          `json:"id"`
	VehicleID      string          `json:"vehicle_id"`
	ClientName     string          `json:"client_name"`
	StartDate      string          `json:"start_date"`
	EndDate        string          `json:"end_date"`
	Place          string          `json:"place"`
	Stage          string          `json:"stage"`
	Days           int             `json:"days"`
	DailyRateCents int64           `json:"daily_rate_cents"`
	TotalCents     int64           `json:"total_cents"`
	Checks         map[string]bool `json:"checks"`
	ExpiresAt      time.Time       `json:"expires_at"`
	CreatedBy      *string         `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Event é uma transição de etapa. UserID nulo indica transição automática (expiração).
type Event struct {
	FromStage *string   `json:"from_stage"`
	ToStage   string    `json:"to_stage"`
	UserID    *string   `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateRequest struct {
	VehicleID  string `json:"vehicle_id"`
	ClientName string `json:"client_name"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Place      string `json:"place"`
}

// NewBooking é o que o repository grava. Diária e total vêm do veículo dentro da transação (RN-02.6).
type NewBooking struct {
	VehicleID  string
	ClientName string
	Period     Period
	Place      string
	CreatedBy  string
}

// Period é o intervalo [Start, End) em datas de calendário.
type Period struct {
	Start time.Time
	End   time.Time
}

func (p Period) Days() int {
	return int(p.End.Sub(p.Start).Hours() / 24)
}

// ParsePeriod valida as datas (RN-02.7) e exige ao menos 1 diária (RN-02.12).
func ParsePeriod(start, end string) (Period, error) {
	s, err := time.Parse(DateLayout, strings.TrimSpace(start))
	if err != nil {
		return Period{}, ErrInvalidDate
	}
	e, err := time.Parse(DateLayout, strings.TrimSpace(end))
	if err != nil {
		return Period{}, ErrInvalidDate
	}
	if !e.After(s) {
		return Period{}, ErrInvalidPeriod
	}
	return Period{Start: s, End: e}, nil
}

func (r CreateRequest) Validate(userID string) (NewBooking, error) {
	n := NewBooking{
		VehicleID:  strings.TrimSpace(r.VehicleID),
		ClientName: strings.TrimSpace(r.ClientName),
		Place:      strings.TrimSpace(r.Place),
		CreatedBy:  userID,
	}
	switch {
	case n.VehicleID == "":
		return n, ErrVehicleRequired
	case n.ClientName == "":
		return n, ErrClientNameRequired
	}
	period, err := ParsePeriod(r.StartDate, r.EndDate)
	if err != nil {
		return n, err
	}
	n.Period = period
	return n, nil
}

// ReservedCheckKeys lista as conferências fixas de todas as etapas. Itens do checklist de
// devolução não podem usar essas chaves.
func ReservedCheckKeys() []string {
	var keys []string
	for _, stage := range []string{StagePreparacao, StageConfirmada, StageEmLocacao, StageDevolvida} {
		keys = append(keys, requiredChecks[stage]...)
	}
	return keys
}

// stageChecks junta as conferências fixas da etapa às extras: em "devolvida", os itens ativos
// do checklist de devolução (RN-01.6).
func stageChecks(stage string, extra []string) ([]string, bool) {
	fixed, ok := requiredChecks[stage]
	if !ok {
		return nil, false
	}
	return append(append([]string{}, fixed...), extra...), true
}

// ValidateChecks aceita só as conferências da etapa atual (fixas e extras).
func ValidateChecks(stage string, checks map[string]bool, extra ...string) error {
	allowed, ok := stageChecks(stage, extra)
	if !ok {
		return ErrStageFinal
	}
	if len(checks) == 0 {
		return ErrInvalidCheck
	}
	for key := range checks {
		if !contains(allowed, key) {
			return fmt.Errorf("%w: %q não pertence à etapa %s", ErrInvalidCheck, key, stage)
		}
	}
	return nil
}

// NextStage devolve a etapa seguinte se todas as conferências da atual, fixas e extras, estiverem feitas (RN-02.4).
func NextStage(stage string, checks map[string]bool, extra ...string) (string, error) {
	next, ok := nextStage[stage]
	if !ok {
		return "", ErrStageFinal
	}
	required, _ := stageChecks(stage, extra)
	var missing []string
	for _, key := range required {
		if !checks[key] {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("%w: %s", ErrChecksPending, strings.Join(missing, ", "))
	}
	return next, nil
}

// Cancelable: só "preparacao" e "confirmada" podem ser canceladas (RN-02.5).
func Cancelable(stage string) bool {
	return stage == StagePreparacao || stage == StageConfirmada
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
