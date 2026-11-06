package photo

import "errors"

var (
	ErrNotFound      = errors.New("foto não encontrada")
	ErrOwnerNotFound = errors.New("veículo ou reserva não encontrado")
	ErrInvalidType   = errors.New("formato inválido: envie JPEG, PNG ou WebP")
	ErrTooLarge      = errors.New("a foto passa de 10 MB")
	ErrEmpty         = errors.New("envie um arquivo de imagem")
	ErrInvalidStage  = errors.New("etapa inválida: use entrega ou devolucao")
	ErrWrongStage    = errors.New("a reserva não está na etapa certa para essa foto")
	ErrInvalidWidth  = errors.New("largura inválida: use 640, 1280 ou 1920")
)
