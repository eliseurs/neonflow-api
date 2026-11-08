package contract

import "errors"

var (
	ErrDocumentNotFound = errors.New("contrato não encontrado")
	ErrVersionNotFound  = errors.New("versão do contrato não encontrada")
	ErrBodyRequired     = errors.New("escreva o texto do contrato")
	ErrBodyTooLong      = errors.New("texto do contrato longo demais")
	ErrLabelTooLong     = errors.New("rótulo da versão longo demais (até 40 caracteres)")
	ErrInvalidClause    = errors.New("informe a cláusula (até 40 caracteres, ex.: 4.2)")
	ErrInvalidKind      = errors.New("tipo do comentário: observacao ou correcao")
	ErrInvalidComment   = errors.New("escreva o comentário (até 10.000 caracteres; redação proposta até 20.000)")
	ErrInvalidAction    = errors.New("ação inválida: enviar, solicitar_ajustes, aprovar, revogar, liberar ou publicar")
	ErrNoteRequired     = errors.New("escreva a justificativa")
	ErrNoteTooLong      = errors.New("justificativa longa demais (até 2.000 caracteres)")
	ErrConfirmRequired  = errors.New("confirme a aprovação expressa desta versão")
	ErrForbiddenAction  = errors.New("seu papel não pode fazer esta ação")
	ErrWrongStatus      = errors.New("a versão não está na situação que permite esta ação")
	ErrHashMismatch     = errors.New("o texto lido não é o desta versão; recarregue e leia de novo")
	ErrForbiddenComment = errors.New("seu papel não comenta contratos")
)
