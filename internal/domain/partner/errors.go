package partner

import "errors"

var (
	ErrPartnerNotFound = errors.New("parceiro não encontrado")
	ErrNameRequired    = errors.New("nome do parceiro é obrigatório")
	ErrInvalidKind     = errors.New("tipo de parceiro inválido: use proprietario, motorista ou indicador (ao menos um)")
	ErrInvalidEmail    = errors.New("e-mail inválido")
	ErrInvalidPercent  = errors.New("percentual deve ficar entre 0 e 100")
	ErrCodeExhausted   = errors.New("não foi possível gerar um código de indicação único; tente de novo")
	// ErrCodeTaken é interno: o código sorteado já existe e o serviço sorteia outro.
	ErrCodeTaken = errors.New("código de indicação já usado")
)
