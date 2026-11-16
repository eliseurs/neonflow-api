// Package siterequest é o pedido feito pelo cliente no site (RN-02.33 a RN-02.37): aluguel ou transfer.
// O pedido não bloqueia o carro. A central confirma (e aí nasce a reserva ou o transfer), recusa ou pede
// ajuste; o cliente acompanha em Minha conta e recebe cada resposta por e-mail.
package siterequest

import (
	"net/mail"
	"regexp"
	"strings"
	"time"

	"neonflow-api/internal/domain/booking"
)

const (
	KindRental   = "aluguel"
	KindTransfer = "transfer"

	// StatusAwaitingEmail: pedido feito sem conta na página do carro, esperando o clique no e-mail (RN-02.38).
	// A central não vê.
	StatusAwaitingEmail = "aguardando_email"
	StatusRequested     = "solicitado"
	StatusAdjustment    = "ajuste_pedido"
	StatusConfirmed     = "confirmado"
	StatusRefused       = "recusado"
	StatusCanceled      = "cancelado"
)

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	nonDigits   = regexp.MustCompile(`\D`)
)

type Request struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	Status       string  `json:"status"`
	CustomerID   string  `json:"customer_id"`
	CustomerName string  `json:"customer_name"`
	Email        string  `json:"email"`
	WhatsApp     string  `json:"whatsapp"`
	Notes        string  `json:"notes"`
	ReferredBy   *string `json:"referred_by_partner_id"`
	// Aluguel, com a estimativa mostrada ao cliente (RN-02.34).
	VehicleID   *string    `json:"vehicle_id"`
	VehicleName string     `json:"vehicle_name"`
	StartAt     *time.Time `json:"start_at"`
	EndAt       *time.Time `json:"end_at"`
	Place       string     `json:"place"`
	KmPlan      string     `json:"km_plan"`
	Quote
	// Transfer.
	Origin      string     `json:"origin"`
	Destination string     `json:"destination"`
	PickupAt    *time.Time `json:"pickup_at"`
	Passengers  *int       `json:"passengers"`
	Flight      string     `json:"flight"`
	// CentralNote é a última mensagem da central: o que ajustar ou o motivo da recusa.
	CentralNote string    `json:"central_note"`
	BookingID   *string   `json:"booking_id"`
	TransferID  *string   `json:"transfer_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Quote é a estimativa do aluguel: diárias, diária, total, caução e km do carro no momento do pedido.
type Quote struct {
	Days           *int   `json:"days"`
	DailyRateCents *int64 `json:"daily_rate_cents"`
	TotalCents     *int64 `json:"total_cents"`
	DepositCents   *int64 `json:"deposit_cents"`
	KmLimitPerDay  *int   `json:"km_limit_per_day"`
	ExtraKmCents   *int64 `json:"extra_km_cents"`
}

// Same diz se dois orçamentos cobram o mesmo: diária, caução e km. As diárias vêm do período, que não muda.
func (q Quote) Same(o Quote) bool {
	return eqI64(q.DailyRateCents, o.DailyRateCents) && eqI64(q.DepositCents, o.DepositCents) &&
		eqInt(q.KmLimitPerDay, o.KmLimitPerDay) && eqI64(q.ExtraKmCents, o.ExtraKmCents)
}

// Detail é o pedido com o histórico.
type Detail struct {
	Request
	Events []Event `json:"events"`
}

type Event struct {
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Note       string    `json:"note"`
	UserID     string    `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// VehicleTerms é o que o carro cobra hoje e o que impede o pedido.
type VehicleTerms struct {
	Name           string
	DailyRateCents int64
	InMaintenance  bool
	DepositCents   *int64
	KmLimitPerDay  *int
	ExtraKmCents   *int64
	MinDays        int
}

// QuoteFor calcula a estimativa como a reserva calcula o total: diária × diárias de 24 horas (RN-02.24),
// com o mínimo de diárias do carro (RN-02.32) e o plano de km (RN-02.18).
func QuoteFor(v VehicleTerms, p booking.Period, kmPlan string) (Quote, error) {
	if v.InMaintenance {
		return Quote{}, booking.ErrVehicleInMaintenance
	}
	days := p.Days()
	if days < v.MinDays {
		return Quote{}, booking.MinDaysError{Min: v.MinDays}
	}
	q := Quote{Days: &days, DailyRateCents: &v.DailyRateCents, DepositCents: v.DepositCents}
	total := v.DailyRateCents * int64(days)
	q.TotalCents = &total
	switch kmPlan {
	case booking.KmPlanLimitado:
		if v.KmLimitPerDay == nil {
			return Quote{}, booking.ErrKmPlanUnavailable
		}
		q.KmLimitPerDay, q.ExtraKmCents = v.KmLimitPerDay, v.ExtraKmCents
	case "":
		q.KmLimitPerDay, q.ExtraKmCents = v.KmLimitPerDay, v.ExtraKmCents
	}
	return q, nil
}

// ClientCar é o carro livre no período, como o cliente vê: sem placa, dono nem dados internos.
type ClientCar struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Seats    int    `json:"seats"`
	Gear     string `json:"gear"`
	Quote
}

// CreateRequest é o pedido do cliente. Kind escolhe quais campos valem.
type CreateRequest struct {
	Kind         string `json:"kind"`
	WhatsApp     string `json:"whatsapp"`
	Notes        string `json:"notes"`
	ReferralCode string `json:"referral_code"`
	VehicleID    string `json:"vehicle_id"`
	StartAt      string `json:"start_at"`
	EndAt        string `json:"end_at"`
	Place        string `json:"place"`
	KmPlan       string `json:"km_plan"`
	Origin       string `json:"origin"`
	Destination  string `json:"destination"`
	PickupAt     string `json:"pickup_at"`
	Passengers   int    `json:"passengers"`
	Flight       string `json:"flight"`
}

// GuestRequest é o pedido feito na página do carro sem conta: só o contato e o aluguel (RN-02.38).
type GuestRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	WhatsApp     string `json:"whatsapp"`
	Notes        string `json:"notes"`
	ReferralCode string `json:"referral_code"`
	VehicleID    string `json:"vehicle_id"`
	StartAt      string `json:"start_at"`
	EndAt        string `json:"end_at"`
	Place        string `json:"place"`
}

// Validate confere o contato e, como pedido de aluguel, o carro e o período.
func (g GuestRequest) Validate(now time.Time) (NewRequest, string, string, error) {
	name := strings.TrimSpace(g.Name)
	email := strings.ToLower(strings.TrimSpace(g.Email))
	if name == "" {
		return NewRequest{}, name, email, ErrNameRequired
	}
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return NewRequest{}, name, email, ErrInvalidEmail
	}
	n, err := CreateRequest{Kind: KindRental, WhatsApp: g.WhatsApp, Notes: g.Notes, ReferralCode: g.ReferralCode,
		VehicleID: g.VehicleID, StartAt: g.StartAt, EndAt: g.EndAt, Place: g.Place}.Validate(now)
	return n, name, email, err
}

// PublicQuote é o orçamento de um carro no período para o visitante: o valor e se o carro está livre.
type PublicQuote struct {
	Available bool `json:"available"`
	Quote
}

// Guest é a conta do contato do pedido sem conta: a que já existe para o e-mail ou uma nova, sem senha.
type Guest struct {
	UserID string
	Name   string
}

// NewRequest é o pedido validado. Quote é preenchido pelo serviço, com o carro de hoje.
type NewRequest struct {
	Kind string
	// Status: vazio é "solicitado"; o pedido sem conta nasce "aguardando_email".
	Status       string
	CustomerID   string
	UserID       string
	WhatsApp     string
	Notes        string
	ReferralCode string
	VehicleID    string
	Period       booking.Period
	Place        string
	KmPlan       string
	Quote        Quote
	Origin       string
	Destination  string
	PickupAt     time.Time
	Passengers   int
	Flight       string
}

// Validate confere os campos do tipo de pedido. Retirada e embarque precisam ser no futuro.
func (r CreateRequest) Validate(now time.Time) (NewRequest, error) {
	n := NewRequest{
		Kind:         strings.TrimSpace(r.Kind),
		WhatsApp:     nonDigits.ReplaceAllString(r.WhatsApp, ""),
		Notes:        strings.TrimSpace(r.Notes),
		ReferralCode: strings.ToUpper(strings.TrimSpace(r.ReferralCode)),
	}
	if len(n.WhatsApp) < 10 || len(n.WhatsApp) > 15 {
		return n, ErrInvalidWhatsApp
	}
	switch n.Kind {
	case KindRental:
		n.VehicleID = strings.TrimSpace(r.VehicleID)
		n.Place = strings.TrimSpace(r.Place)
		n.KmPlan = strings.ToLower(strings.TrimSpace(r.KmPlan))
		if !uuidPattern.MatchString(n.VehicleID) {
			return n, booking.ErrVehicleRequired
		}
		if n.KmPlan != "" && n.KmPlan != booking.KmPlanLivre && n.KmPlan != booking.KmPlanLimitado {
			return n, booking.ErrInvalidKmPlan
		}
		p, err := booking.ParsePeriod(r.StartAt, r.EndAt)
		if err != nil {
			return n, err
		}
		if !p.Start.After(now) {
			return n, ErrPastDate
		}
		n.Period = p
	case KindTransfer:
		n.Origin = strings.TrimSpace(r.Origin)
		n.Destination = strings.TrimSpace(r.Destination)
		n.Flight = strings.TrimSpace(r.Flight)
		n.Passengers = r.Passengers
		if n.Origin == "" || n.Destination == "" {
			return n, ErrRouteRequired
		}
		if n.Passengers < 1 {
			return n, ErrInvalidPassengers
		}
		t, err := booking.ParseInstant(r.PickupAt)
		if err != nil {
			return n, err
		}
		if !t.After(now) {
			return n, ErrPastDate
		}
		n.PickupAt = t
	default:
		return n, ErrInvalidKind
	}
	return n, nil
}

// ConfirmRequest: no transfer, a central informa o preço (RN-02.35); no aluguel, nada.
type ConfirmRequest struct {
	PriceCents int64 `json:"price_cents"`
}

// NoteRequest é a mensagem da central ao pedir ajuste ou recusar.
type NoteRequest struct {
	Note string `json:"note"`
}

// Account é a conta do cliente logado e o cadastro dela.
type Account struct {
	UserID     string
	CustomerID string
	Name       string
}

func eqI64(a, b *int64) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
func eqInt(a, b *int) bool   { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
