package user

import "errors"

var (
	// ErrSelfLockout evita que o admin perca o próprio acesso (e deixe o painel sem admin).
	ErrSelfLockout   = errors.New("você não pode desativar a si mesmo nem tirar o seu papel de admin")
	ErrWrongPassword = errors.New("a senha atual não confere")
)
