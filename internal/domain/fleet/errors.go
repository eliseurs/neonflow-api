package fleet

import "errors"

var (
	ErrVehicleNotFound  = errors.New("veículo não encontrado")
	ErrPlateTaken       = errors.New("placa já cadastrada")
	ErrInvalidPlate     = errors.New("placa inválida: use o formato ABC1234 ou ABC1D23")
	ErrNameRequired     = errors.New("nome do veículo é obrigatório")
	ErrInvalidCategory  = errors.New("categoria inválida: use economy ou premium")
	ErrOwnerRequired    = errors.New("dono do veículo é obrigatório")
	ErrInvalidSeats     = errors.New("número de lugares deve ser maior que zero")
	ErrInvalidGear      = errors.New("câmbio inválido: use manual ou automatico")
	ErrInvalidDailyRate = errors.New("diária deve ser maior que zero, em centavos")
)
