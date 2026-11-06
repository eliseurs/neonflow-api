package customer

import "errors"

var (
	ErrNotFound        = errors.New("cliente não encontrado")
	ErrNameRequired    = errors.New("nome do cliente é obrigatório")
	ErrInvalidWhatsApp = errors.New("WhatsApp inválido: use o número com DDI e DDD, ex.: +55 54 99999-0000")
	ErrInvalidEmail    = errors.New("e-mail inválido")
	ErrInvalidLanguage = errors.New("idioma inválido: use pt ou en")
	ErrWhatsAppTaken   = errors.New("já existe um cliente com esse WhatsApp")
	ErrAnonymized      = errors.New("cliente anonimizado: os dados não podem ser alterados")
)
