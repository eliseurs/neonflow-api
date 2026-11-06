package payout

import "errors"

var (
	ErrNotFound         = errors.New("pagamento ao parceiro não encontrado")
	ErrAlreadyPaid      = errors.New("pagamento já está marcado como pago")
	ErrPercentUndefined = errors.New("o parceiro não tem o percentual definido; defina em Parceiros antes de pagar")
	ErrInvalidFilter    = errors.New("filtro inválido: tipo comissao, repasse ou motorista, situação pendente ou paga")
)
