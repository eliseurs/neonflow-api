package account

import "errors"

var (
	ErrInvalidToken        = errors.New("link inválido ou vencido; peça um novo")
	ErrSendFailed          = errors.New("não foi possível enviar o e-mail; tente de novo em instantes")
	ErrPartnerNotFound     = errors.New("parceiro não encontrado")
	ErrPartnerInactive     = errors.New("parceiro desativado: reative antes de convidar")
	ErrPartnerHasAccount   = errors.New("este parceiro já tem conta")
	ErrPartnerWithoutEmail = errors.New("cadastre um e-mail válido do parceiro antes de convidar")
)
