package booking

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"neonflow-api/internal/pkg/timezone"
)

// Etapas da locação (RN-02.4). Ativas são as que bloqueiam o período: todas, menos encerrada e cancelada.
const (
	StagePreparacao = "preparacao"
	StageConfirmada = "confirmada"
	StageEmLocacao  = "em_locacao"
	StageDevolvida  = "devolvida"
	StageEncerrada  = "encerrada"
	StageCancelada  = "cancelada"

	// DateLayout é o formato das datas (AAAA-MM-DD), aceito por compatibilidade: vira 09:00 de Gramado.
	DateLayout = "2006-01-02"
	// DateTimeLayout é a data e hora local de Gramado, sem fuso (como o campo datetime-local do navegador).
	DateTimeLayout = "2006-01-02T15:04"
	// LegacyHour é a hora dada a uma data sem horário: a do protótipo e a da migração V5.
	LegacyHour = 9
	// DayLength é a diária: blocos de 24 horas a partir da retirada (RN-02.24).
	DayLength = 24 * time.Hour
	// LateTolerance é a tolerância na devolução: até 1 hora depois do combinado não cobra (RN-02.19).
	LateTolerance = time.Hour
	// Plano de quilometragem escolhido na reserva (RN-02.18). Vazio: limitado se o veículo tiver limite.
	KmPlanLivre    = "livre"
	KmPlanLimitado = "limitado"

	// ClockSkew é a folga aceita para uma hora informada no futuro (relógios desencontrados).
	ClockSkew = 5 * time.Minute

	// PaymentWindow é o prazo para pagar a pré-reserva; depois ela é cancelada (RN-02.8, RN-02.11, ADR-0012).
	PaymentWindow = 24 * time.Hour
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

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
	StartAt        time.Time       `json:"start_at"`
	EndAt          time.Time       `json:"end_at"`
	StartDate      string          `json:"start_date"` // dia de StartAt em Gramado; mantido por compatibilidade
	EndDate        string          `json:"end_date"`   // dia de EndAt em Gramado; mantido por compatibilidade
	Place          string          `json:"place"`
	Stage          string          `json:"stage"`
	Days           int             `json:"days"`
	DailyRateCents int64           `json:"daily_rate_cents"`
	TotalCents     int64           `json:"total_cents"`
	Checks         map[string]bool `json:"checks"`
	// Hora real da entrega e da devolução, informadas pela central (RN-02.19).
	DeliveredAt *time.Time `json:"delivered_at"`
	ReturnedAt  *time.Time `json:"returned_at"`
	// Minutos de atraso na devolução e o valor deles: diária ÷ 1440 por minuto (RN-02.19).
	ExtraMinutes int   `json:"extra_minutes"`
	ExtraCents   int64 `json:"extra_cents"`
	// Caução e plano de km copiados do veículo na criação (RN-06.4, RN-02.18). KmLimitPerDay nulo = km livre.
	DepositCents  *int64 `json:"deposit_cents"`
	KmLimitPerDay *int   `json:"km_limit_per_day"`
	ExtraKmCents  *int64 `json:"extra_km_cents"`
	// Odômetro na entrega e na devolução e o excedente cobrado (RN-02.18).
	// Parceiro que indicou o cliente (origem, RN-07.5). A comissão vem na fase 2 (RN-03.2).
	ReferredByPartnerID *string   `json:"referred_by_partner_id"`
	DeliveredKm         *int      `json:"delivered_km"`
	ReturnedKm          *int      `json:"returned_km"`
	ExcessKm            int       `json:"excess_km"`
	ExcessKmCents       int64     `json:"excess_km_cents"`
	ExpiresAt           time.Time `json:"expires_at"`
	CreatedBy           *string   `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Event é uma transição de etapa. UserID nulo indica transição automática (expiração).
type Event struct {
	FromStage *string   `json:"from_stage"`
	ToStage   string    `json:"to_stage"`
	UserID    *string   `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateRequest aceita start_at e end_at (data e hora). start_date e end_date (só data) continuam
// aceitos por compatibilidade e viram 09:00.
type CreateRequest struct {
	VehicleID  string `json:"vehicle_id"`
	ClientName string `json:"client_name"`
	StartAt    string `json:"start_at"`
	EndAt      string `json:"end_at"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Place      string `json:"place"`
	KmPlan     string `json:"km_plan"`
	// ReferredByPartnerID é o parceiro que indicou (vem do código na mensagem do WhatsApp).
	ReferredByPartnerID string `json:"referred_by_partner_id"`
}

// AdvanceRequest é o corpo opcional do avanço de etapa. At é a hora real da entrega
// (confirmada → em locação) ou da devolução (em locação → devolvida); vazia, vale agora.
type AdvanceRequest struct {
	At string `json:"at"`
	// Km é o odômetro na entrega ou na devolução; obrigatório quando o plano é limitado (RN-02.18).
	Km *int `json:"km"`
}

// Record é o que a transição grava além da etapa. Campos nulos não mudam a reserva.
type Record struct {
	DeliveredAt   *time.Time
	ReturnedAt    *time.Time
	ExtraMinutes  *int
	ExtraCents    *int64
	DeliveredKm   *int
	ReturnedKm    *int
	ExcessKm      *int
	ExcessKmCents *int64
}

// ExcessKm é o que passou da franquia: limite por dia × diárias contratadas. As horas de atraso não
// trazem franquia, como na Localiza (RN-02.18, q90). Abaixo da franquia, zero.
func ExcessKm(limitPerDay, contractedDays, driven int) int {
	return max(driven-limitPerDay*contractedDays, 0)
}

// minutesPerDay é a diária em minutos: o atraso é cobrado a diária ÷ 1440 por minuto.
const minutesPerDay = 24 * 60

// LateMinutes conta os minutos exatos de atraso de uma devolução: nada até 1 hora depois do combinado;
// passado isso, todos os minutos completos desde o horário combinado (RN-02.19). Devolver antes segue RN-02.28.
func LateMinutes(plannedEnd, returned time.Time) int {
	late := returned.Sub(plannedEnd)
	if late <= LateTolerance {
		return 0
	}
	return int(late / time.Minute)
}

// LateCents é o valor do atraso: diária ÷ 1440 por minuto, arredondado ao centavo. A cada 24 horas
// completa-se uma diária, então o valor nunca passa de uma diária por dia de atraso (RN-02.19, q89).
func LateCents(minutes int, dailyRateCents int64) int64 {
	return (int64(minutes)*dailyRateCents + minutesPerDay/2) / minutesPerDay
}

// NewBooking é o que o repository grava. Diária e total vêm do veículo dentro da transação (RN-02.6).
type NewBooking struct {
	VehicleID  string
	ClientName string
	Period     Period
	Place      string
	CreatedBy  string
	KmPlan     string
	ReferredBy string
}

// Period é o intervalo [Start, End) entre a retirada e a devolução combinadas.
type Period struct {
	Start time.Time
	End   time.Time
}

// Days conta as diárias em blocos de 24 horas a partir da retirada, arredondando para cima:
// 24h = 1 diária, 25h = 2 (RN-02.24). Qualquer período vale ao menos 1 diária (RN-02.12).
// A tolerância de 1 hora (RN-02.19) vale para o atraso na devolução, não para o período combinado.
func (p Period) Days() int {
	d := p.End.Sub(p.Start)
	days := int(d / DayLength)
	if d%DayLength > 0 {
		days++
	}
	return max(days, 1)
}

// ParseInstant lê a retirada ou a devolução. Aceita, nesta ordem: RFC 3339 com fuso
// (2026-12-10T14:00:00-03:00), data e hora de Gramado (2026-12-10T14:00, com ou sem segundos)
// e só a data (2026-12-10), que vira 09:00 por compatibilidade.
func ParseInstant(value string) (time.Time, error) {
	v := strings.TrimSpace(value)
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	for _, layout := range []string{DateTimeLayout, DateTimeLayout + ":05"} {
		if t, err := time.ParseInLocation(layout, v, timezone.Location); err == nil {
			return t, nil
		}
	}
	if d, err := time.ParseInLocation(DateLayout, v, timezone.Location); err == nil {
		return d.Add(LegacyHour * time.Hour), nil
	}
	return time.Time{}, ErrInvalidDate
}

// ParsePeriod valida a retirada e a devolução (RN-02.7, RN-02.23): a devolução vem depois da retirada.
func ParsePeriod(start, end string) (Period, error) {
	s, err := ParseInstant(start)
	if err != nil {
		return Period{}, err
	}
	e, err := ParseInstant(end)
	if err != nil {
		return Period{}, err
	}
	if !e.After(s) {
		return Period{}, ErrInvalidPeriod
	}
	return Period{Start: s, End: e}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (r CreateRequest) Validate(userID string) (NewBooking, error) {
	n := NewBooking{
		VehicleID:  strings.TrimSpace(r.VehicleID),
		ClientName: strings.TrimSpace(r.ClientName),
		Place:      strings.TrimSpace(r.Place),
		CreatedBy:  userID,
		KmPlan:     strings.ToLower(strings.TrimSpace(r.KmPlan)),
		ReferredBy: strings.TrimSpace(r.ReferredByPartnerID),
	}
	if n.ReferredBy != "" && !uuidPattern.MatchString(n.ReferredBy) {
		return n, ErrReferrerNotFound
	}
	if n.KmPlan != "" && n.KmPlan != KmPlanLivre && n.KmPlan != KmPlanLimitado {
		return n, ErrInvalidKmPlan
	}
	switch {
	case n.VehicleID == "":
		return n, ErrVehicleRequired
	case n.ClientName == "":
		return n, ErrClientNameRequired
	}
	period, err := ParsePeriod(firstNonEmpty(r.StartAt, r.StartDate), firstNonEmpty(r.EndAt, r.EndDate))
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
