package fleet

import "errors"

var (
	ErrVehicleNotFound      = errors.New("veículo não encontrado")
	ErrPlateTaken           = errors.New("placa já cadastrada")
	ErrInvalidPlate         = errors.New("placa inválida: use o formato ABC1234 ou ABC1D23")
	ErrNameRequired         = errors.New("nome do veículo é obrigatório")
	ErrInvalidCategory      = errors.New("categoria inválida: use economy ou premium")
	ErrOwnerRequired        = errors.New("dono do veículo é obrigatório")
	ErrInvalidSeats         = errors.New("número de lugares deve ser maior que zero")
	ErrInvalidGear          = errors.New("câmbio inválido: use manual ou automatico")
	ErrInvalidDailyRate     = errors.New("diária deve ser maior que zero, em centavos")
	ErrInvalidDeposit       = errors.New("caução deve ser maior que zero, em centavos")
	ErrInvalidKmPlan        = errors.New("limite de km e valor do km excedente devem ser maiores que zero")
	ErrKmPlanIncomplete     = errors.New("o valor do km excedente exige o limite de km por dia")
	ErrOwnerPartnerNotFound = errors.New("parceiro proprietário não encontrado")
	ErrInvalidYear          = errors.New("ano inválido")
	ErrInvalidFuel          = errors.New("combustível inválido: flex, gasolina, etanol, diesel, elétrico ou híbrido")
	ErrInvalidDoors         = errors.New("portas: de 2 a 5")
	ErrInvalidFeature       = errors.New("item desconhecido na lista do carro")
	ErrDescriptionTooLong   = errors.New("descrição longa demais (até 2000 caracteres)")
)
