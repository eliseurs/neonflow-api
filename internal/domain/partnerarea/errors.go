package partnerarea

import "errors"

// ErrNoPartner: a conta logada não é de um parceiro ativo.
var ErrNoPartner = errors.New("entre com a sua conta de parceiro")

var (
	ErrNotDriver      = errors.New("só motoristas aceitam corridas")
	ErrNotYourVehicle = errors.New("escolha um dos seus carros ou deixe sem carro")
	ErrNotYourRide    = errors.New("corrida não encontrada entre as suas")
)
