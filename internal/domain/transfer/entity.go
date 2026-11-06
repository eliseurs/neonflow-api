// Package transfer é o transfer operado pela central: corrida com preço sob consulta; a central define o
// valor para o motorista, chama os motoristas um a um pelo WhatsApp e marca quem aceitou (RN-02.9 a 02.31).
package transfer

import (
	"fmt"
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
	// Valor para o motorista enquanto ninguém aceitou (RN-02.22: só aumenta). Nulo = ainda não definido.
	OfferCents *int64 `json:"offer_cents"`
	// Motorista que aceitou e o valor aceito (RN-03.14). Nem sempre é o dono do carro.
	DriverPartnerID *string `json:"driver_partner_id"`
	DriverName      *string `json:"driver_name"`
	DriverCents     *int64  `json:"driver_cents"`
	// Carro usado na corrida, quando registrado (RN-03.10 revista).
	VehicleID   *string    `json:"vehicle_id"`
	VehicleName *string    `json:"vehicle_name"`
	PaidAt      *time.Time `json:"paid_at"`
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

// OfferRequest define ou aumenta o valor para o motorista (RN-02.22).
type OfferRequest struct {
	OfferCents int64 `json:"offer_cents"`
}

// ValidateOffer: valor positivo e que não diminui (RN-02.22).
func ValidateOffer(cents int64, current *int64) error {
	switch {
	case cents <= 0:
		return ErrInvalidOffer
	case current != nil && cents < *current:
		return ErrOfferDecreased
	}
	return nil
}

// NotifyRequest registra que a central chamou um motorista pelo WhatsApp.
type NotifyRequest struct {
	DriverPartnerID string `json:"driver_partner_id"`
}

// AcceptRequest marca o motorista que aceitou (RN-02.17). OfferCents, se vier, vale como o valor final
// (não menor que o atual); VehicleID é o carro da corrida, opcional.
type AcceptRequest struct {
	DriverPartnerID string `json:"driver_partner_id"`
	OfferCents      *int64 `json:"offer_cents"`
	VehicleID       string `json:"vehicle_id"`
}

// Validate normaliza e confere o formato; motorista e carro são conferidos no banco.
func (r AcceptRequest) Validate(current *int64) (AcceptRequest, error) {
	r.DriverPartnerID = strings.TrimSpace(r.DriverPartnerID)
	r.VehicleID = strings.TrimSpace(r.VehicleID)
	switch {
	case !uuidPattern.MatchString(r.DriverPartnerID):
		return r, ErrDriverNotFound
	case r.VehicleID != "" && !uuidPattern.MatchString(r.VehicleID):
		return r, ErrVehicleNotFound
	case r.OfferCents == nil && current == nil:
		return r, ErrNoOffer
	case r.OfferCents != nil:
		if err := ValidateOffer(*r.OfferCents, current); err != nil {
			return r, err
		}
	}
	return r, nil
}

// Driver é quem pode fazer a corrida: parceiro ativo motorista ou proprietário (o dono pode dirigir o
// próprio carro, mas nem sempre quem dirige é o dono).
type Driver struct {
	PartnerID string   `json:"partner_id"`
	Name      string   `json:"name"`
	WhatsApp  string   `json:"whatsapp"`
	Kinds     []string `json:"kinds"`
	// Outras corridas do motorista no mesmo dia (horário de Gramado), para não atribuir duas juntas.
	SameDay []time.Time `json:"same_day"`
	// Carros de que ele é dono, para sugerir o carro da corrida.
	Vehicles []DriverVehicle `json:"vehicles"`
}

type DriverVehicle struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Plate string `json:"plate"`
}

// FormatReais escreve centavos como "R$ 1.350,00" para o histórico.
func FormatReais(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole := fmt.Sprintf("%d", cents/100)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "." + whole[i:]
	}
	return fmt.Sprintf("%sR$ %s,%02d", sign, whole, cents%100)
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
