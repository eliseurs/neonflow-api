package terms

import "errors"

var (
	ErrNoTerms         = errors.New("os termos ainda não foram publicados")
	ErrBodyRequired    = errors.New("escreva o texto dos termos")
	ErrBodyTooLong     = errors.New("texto dos termos longo demais")
	ErrBookingNotFound = errors.New("reserva não encontrada")
	ErrWrongStage      = errors.New("o aceite é enviado antes da retirada (pré-reserva ou confirmada)")
	ErrLinkNotFound    = errors.New("link inválido ou vencido; peça um novo à central")
	ErrInvalidName     = errors.New("digite o seu nome completo")
	ErrVersionChanged  = errors.New("os termos mudaram; recarregue a página e leia de novo")
)
