// Package payout são os pagamentos a parceiros, gerados quando a reserva é encerrada ou o transfer é
// concluído (pagos e com o serviço finalizado) e marcados como pagos pela central:
//   - comissão do indicador: diárias da reserva ou preço do transfer, saindo da parte do Neon Flow (RN-03.12, RN-03.14);
//   - repasse ao dono do carro parceiro, sobre diárias + atraso + km excedente (RN-03.13);
//   - motorista do transfer: valor da oferta aceita + espera extra (RN-03.9, RN-02.16).
package payout

import "time"

const (
	KindCommission = "comissao"
	KindOwnerShare = "repasse"
	KindDriver     = "motorista"

	StatusPending = "pendente"
	StatusPaid    = "paga"
)

type Payout struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	// Origem: uma reserva de aluguel ou um transfer.
	BookingID  *string `json:"booking_id"`
	TransferID *string `json:"transfer_id"`
	ClientName string  `json:"client_name"`
	// Carro da reserva ou "origem → destino" do transfer.
	Description string `json:"description"`
	// Fim da reserva ou hora do transfer, para agrupar por mês.
	EndAt       time.Time `json:"end_at"`
	PartnerID   string    `json:"partner_id"`
	PartnerName string    `json:"partner_name"`
	BaseCents   int64     `json:"base_cents"`
	// Percentual copiado do parceiro; nulo = a definir (é copiado de novo no pagamento).
	Percent     *float64   `json:"percent"`
	AmountCents *int64     `json:"amount_cents"`
	Status      string     `json:"status"`
	PaidAt      *time.Time `json:"paid_at"`
	PaidByName  *string    `json:"paid_by_name"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Filter da listagem; campos vazios não filtram.
type Filter struct {
	Kind      string
	Status    string
	PartnerID string
}

// Validate recusa tipo ou situação desconhecidos.
func (f Filter) Validate() error {
	if f.Kind != "" && f.Kind != KindCommission && f.Kind != KindOwnerShare && f.Kind != KindDriver {
		return ErrInvalidFilter
	}
	if f.Status != "" && f.Status != StatusPending && f.Status != StatusPaid {
		return ErrInvalidFilter
	}
	return nil
}

// Amount é base × pontos-base ÷ 10000, arredondado ao centavo.
func Amount(baseCents int64, bp int) int64 {
	return (baseCents*int64(bp) + 5000) / 10000
}
