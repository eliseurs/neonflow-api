// Package transfer é o transfer operado pela central: corrida com preço sob consulta, oferecida aos
// motoristas parceiros com valor e prazo, aceita pelo primeiro, paga e concluída (RN-02.9 a 02.31).
package transfer

import (
	"regexp"
	"strings"
	"time"

	"neonflow-api/internal/pkg/timezone"
)

const (
	StageWaiting   = "aguardando_motorista"
	StageAssigned  = "com_motorista"
	StageOnTrip    = "em_andamento"
	StageDone      = "concluido"
	StageCancelled = "cancelado"

	// DateTimeLayout é a data e hora local de Gramado, sem fuso (como o datetime-local do navegador).
	DateTimeLayout = "2006-01-02T15:04"
	// FreeCancelWindow: cancelamento grátis até 24 horas antes; depois, 50% do valor (RN-02.10).
	FreeCancelWindow = 24 * time.Hour
	// MaxOfferMinutes limita o prazo de uma oferta a 7 dias.
	MaxOfferMinutes = 7 * 24 * 60
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Transfer struct {
	ID          string    `json:"id"`
	ClientName  string    `json:"client_name"`
	PickupAt    time.Time `json:"pickup_at"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	Flight      string    `json:"flight"`
	Passengers  int       `json:"passengers"`
	Notes       string    `json:"notes"`
	PriceCents  int64     `json:"price_cents"`
	Stage       string    `json:"stage"`
	// Oferta aberta aos motoristas: valor e fim do prazo (RN-02.17, RN-02.30). Nulos = sem oferta.
	OfferCents     *int64     `json:"offer_cents"`
	OfferExpiresAt *time.Time `json:"offer_expires_at"`
	// Motorista que aceitou e o valor da oferta aceita (RN-03.9).
	DriverPartnerID *string    `json:"driver_partner_id"`
	DriverName      *string    `json:"driver_name"`
	DriverCents     *int64     `json:"driver_cents"`
	PaidAt          *time.Time `json:"paid_at"`
	// Espera no aeroporto além dos 60 minutos inclusos, informada na conclusão; vai para o motorista (RN-02.16).
	ExtraWaitCents      int64     `json:"extra_wait_cents"`
	CancelFeeCents      int64     `json:"cancel_fee_cents"`
	FlightCancelled     bool      `json:"flight_cancelled"`
	ReferredByPartnerID *string   `json:"referred_by_partner_id"`
	CreatedBy           *string   `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Event struct {
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	UserName  *string   `json:"user_name"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateRequest é o pedido registrado pela central depois de combinar o preço com o cliente.
type CreateRequest struct {
	ClientName          string `json:"client_name"`
	PickupAt            string `json:"pickup_at"`
	Origin              string `json:"origin"`
	Destination         string `json:"destination"`
	Flight              string `json:"flight"`
	Passengers          int    `json:"passengers"`
	Notes               string `json:"notes"`
	PriceCents          int64  `json:"price_cents"`
	ReferredByPartnerID string `json:"referred_by_partner_id"`
}

// NewTransfer é o que o repository grava.
type NewTransfer struct {
	ClientName  string
	PickupAt    time.Time
	Origin      string
	Destination string
	Flight      string
	Passengers  int
	Notes       string
	PriceCents  int64
	ReferredBy  string
	CreatedBy   string
}

func (r CreateRequest) Validate(userID string) (NewTransfer, error) {
	n := NewTransfer{
		ClientName:  strings.TrimSpace(r.ClientName),
		Origin:      strings.TrimSpace(r.Origin),
		Destination: strings.TrimSpace(r.Destination),
		Flight:      strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(r.Flight), " ", "")),
		Passengers:  r.Passengers,
		Notes:       strings.TrimSpace(r.Notes),
		PriceCents:  r.PriceCents,
		ReferredBy:  strings.TrimSpace(r.ReferredByPartnerID),
		CreatedBy:   userID,
	}
	if n.Passengers == 0 {
		n.Passengers = 1
	}
	at, err := time.ParseInLocation(DateTimeLayout, strings.TrimSpace(r.PickupAt), timezone.Location)
	switch {
	case n.ClientName == "":
		return n, ErrClientRequired
	case err != nil:
		return n, ErrInvalidPickup
	case n.Origin == "" || n.Destination == "":
		return n, ErrRouteRequired
	case n.Passengers < 1 || n.Passengers > 60:
		return n, ErrInvalidPassengers
	case n.PriceCents <= 0:
		return n, ErrInvalidPrice
	case n.ReferredBy != "" && !uuidPattern.MatchString(n.ReferredBy):
		return n, ErrReferrerNotFound
	}
	n.PickupAt = at
	return n, nil
}

// OfferRequest abre ou reabre a oferta aos motoristas.
type OfferRequest struct {
	OfferCents int64 `json:"offer_cents"`
	Minutes    int   `json:"minutes"`
}

// Validate: valor positivo e que não diminui ao reabrir (RN-02.22); prazo de 1 minuto a 7 dias (RN-02.30).
func (r OfferRequest) Validate(current *int64) error {
	switch {
	case r.OfferCents <= 0:
		return ErrInvalidOffer
	case current != nil && r.OfferCents < *current:
		return ErrOfferDecreased
	case r.Minutes < 1 || r.Minutes > MaxOfferMinutes:
		return ErrInvalidDeadline
	}
	return nil
}

// AcceptRequest registra o motorista que aceitou primeiro (RN-02.17).
type AcceptRequest struct {
	DriverPartnerID string `json:"driver_partner_id"`
}

// CompleteRequest conclui a corrida com a espera extra, em reais, combinada com o cliente (RN-02.16).
type CompleteRequest struct {
	ExtraWaitCents int64 `json:"extra_wait_cents"`
}

// CancelRequest informa se a companhia aérea cancelou o voo (sem cobrança, RN-02.10).
type CancelRequest struct {
	FlightCancelled bool `json:"flight_cancelled"`
}

// CancelFee: grátis até 24 horas antes do serviço ou com voo cancelado pela companhia; dentro das
// 24 horas, 50% do preço, arredondado ao centavo (RN-02.10).
func CancelFee(priceCents int64, pickupAt, now time.Time, flightCancelled bool) int64 {
	if flightCancelled || pickupAt.Sub(now) >= FreeCancelWindow {
		return 0
	}
	return (priceCents + 1) / 2
}

// Cancellable: depois de começar a corrida, não se cancela.
func Cancellable(stage string) bool {
	return stage == StageWaiting || stage == StageAssigned
}
