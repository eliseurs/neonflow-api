package siterequest

import "errors"

var (
	ErrNotFound          = errors.New("pedido não encontrado")
	ErrInvalidKind       = errors.New("tipo de pedido inválido: use aluguel ou transfer")
	ErrInvalidWhatsApp   = errors.New("informe o WhatsApp com DDI e DDD, só números")
	ErrPastDate          = errors.New("a data do pedido precisa ser no futuro")
	ErrRouteRequired     = errors.New("informe a origem e o destino")
	ErrInvalidPassengers = errors.New("informe quantos passageiros")
	ErrNoAccount         = errors.New("entre com a sua conta de cliente para pedir")
	ErrStatusChanged     = errors.New("o pedido mudou de situação; recarregue e tente de novo")
	ErrKindChanged       = errors.New("o ajuste não muda o tipo do pedido")
	ErrNoteRequired      = errors.New("escreva a mensagem para o cliente")
	ErrPriceRequired     = errors.New("informe o preço do transfer para confirmar")
	ErrPriceChanged      = errors.New("o valor do carro mudou desde o pedido; peça ajuste ao cliente")
	ErrNameRequired      = errors.New("informe o seu nome")
	ErrSendFailed        = errors.New("não foi possível enviar o e-mail; tente de novo em instantes")
	ErrInvalidEmail      = errors.New("informe um e-mail válido")
	// ErrEmailInUse: o e-mail é de um parceiro, da equipe ou de uma conta desativada.
	ErrEmailInUse = errors.New("este e-mail já é usado por outro acesso; use outro e-mail ou fale com a central pelo WhatsApp")
)
