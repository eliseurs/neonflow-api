package transfer

import "errors"

var (
	ErrNotFound          = errors.New("transfer não encontrado")
	ErrClientRequired    = errors.New("nome do cliente é obrigatório")
	ErrInvalidPickup     = errors.New("data e hora inválidas: use AAAA-MM-DDTHH:MM")
	ErrRouteRequired     = errors.New("informe a origem e o destino")
	ErrInvalidPassengers = errors.New("passageiros deve ficar entre 1 e 60")
	ErrInvalidPrice      = errors.New("o preço da corrida deve ser maior que zero")
	ErrReferrerNotFound  = errors.New("parceiro indicador não encontrado")
	ErrInvalidOffer      = errors.New("o valor da oferta deve ser maior que zero")
	ErrOfferDecreased    = errors.New("a oferta reaberta não pode ser menor que a anterior")
	ErrInvalidDeadline   = errors.New("o prazo da oferta deve ficar entre 1 minuto e 7 dias")
	ErrNoOffer           = errors.New("abra a oferta aos motoristas antes de registrar o aceite")
	ErrDriverNotFound    = errors.New("motorista não encontrado ou inativo")
	ErrWrongStage        = errors.New("a corrida não está na etapa certa para essa ação")
	ErrNotPaid           = errors.New("marque o pagamento do cliente antes de concluir")
	ErrInvalidExtraWait  = errors.New("a espera extra não pode ser negativa")
)
