package checklist

import "errors"

var (
	ErrItemNotFound  = errors.New("item do checklist não encontrado")
	ErrLabelRequired = errors.New("nome do item é obrigatório")
	ErrInvalidLabel  = errors.New("o nome do item precisa ter letras ou números")
	ErrKeyTaken      = errors.New("já existe um item com esse nome")
	ErrKeyReserved   = errors.New("esse nome é usado por uma conferência fixa da reserva; escolha outro")
)
