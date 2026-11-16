// Package partnerarea é a área do parceiro logado (RN-07.8 a RN-07.11), só leitura: o proprietário vê os
// carros e as reservas deles sem dados do cliente, o motorista vê as corridas, e todo parceiro vê as
// indicações e o próprio código. Cada um vê o que recebe, com a base e o percentual.
package partnerarea

import "time"

type Partner struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Kinds        []string `json:"kinds"`
	ReferralCode string   `json:"referral_code"`
}

// Has diz se o parceiro tem o tipo (proprietario, motorista, indicador).
func (p Partner) Has(kind string) bool {
	for _, k := range p.Kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// Payout é o que o parceiro recebe: base, percentual (nulo = a definir ou valor fixo), valor e se já foi pago.
type Payout struct {
	BaseCents   int64      `json:"base_cents"`
	PercentBP   *int       `json:"percent_bp"`
	AmountCents *int64     `json:"amount_cents"`
	PaidAt      *time.Time `json:"paid_at"`
}

type Vehicle struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Plate         string `json:"plate"`
	InMaintenance bool   `json:"in_maintenance"`
}

// Booking é a reserva do carro do proprietário, sem nome nem contato do cliente (RN-07.9).
type Booking struct {
	ID           string    `json:"id"`
	VehicleName  string    `json:"vehicle_name"`
	StartAt      time.Time `json:"start_at"`
	EndAt        time.Time `json:"end_at"`
	Stage        string    `json:"stage"`
	Days         int       `json:"days"`
	ExtraMinutes int       `json:"extra_minutes"`
	ExcessKm     int       `json:"excess_km"`
	Payout       *Payout   `json:"payout"`
}

// Ride é a corrida do motorista, com o necessário para executar (RN-07.10). O contato fica com a central.
type Ride struct {
	ID          string    `json:"id"`
	PickupAt    time.Time `json:"pickup_at"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	Flight      string    `json:"flight"`
	Passengers  int       `json:"passengers"`
	ClientName  string    `json:"client_name"`
	Stage       string    `json:"stage"`
	DriverCents int64     `json:"driver_cents"`
	Payout      *Payout   `json:"payout"`
}

// Referral é uma indicação do parceiro: serviço, data, situação e comissão, sem o cliente (RN-07.11).
type Referral struct {
	ID     string    `json:"id"`
	Kind   string    `json:"kind"` // aluguel ou transfer
	At     time.Time `json:"at"`
	Stage  string    `json:"stage"`
	Payout *Payout   `json:"payout"`
}

// Area é tudo o que o parceiro vê; as seções que não são do tipo dele vêm vazias.
type Area struct {
	Partner   Partner    `json:"partner"`
	Vehicles  []Vehicle  `json:"vehicles"`
	Bookings  []Booking  `json:"bookings"`
	Rides     []Ride     `json:"rides"`
	Referrals []Referral `json:"referrals"`
}

const (
	KindOwner  = "proprietario"
	KindDriver = "motorista"
)
