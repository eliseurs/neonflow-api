package maintenance

import "errors"

var (
	ErrNotFound            = errors.New("manutenção não encontrada")
	ErrVehicleNotFound     = errors.New("veículo não encontrado")
	ErrInvalidDate         = errors.New("data inválida: use AAAA-MM-DD")
	ErrFutureDate          = errors.New("a data da manutenção não pode estar no futuro")
	ErrInvalidKind         = errors.New("tipo inválido: preventiva, corretiva, revisão, pneus ou outros")
	ErrDescriptionRequired = errors.New("descreva o que foi feito")
	ErrDescriptionTooLong  = errors.New("descrição longa demais (até 1000 caracteres)")
	ErrInvalidCost         = errors.New("o custo não pode ser negativo")
	ErrInvalidOdometer     = errors.New("a quilometragem não pode ser negativa")
)
