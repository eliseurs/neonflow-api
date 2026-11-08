package authentication

import "errors"

var (
	ErrInvalidCredentials = errors.New("e-mail ou senha inválidos")
	ErrUserNotFound       = errors.New("usuário não encontrado")
	ErrEmailTaken         = errors.New("e-mail já cadastrado")
	ErrNameRequired       = errors.New("nome é obrigatório")
	ErrInvalidEmail       = errors.New("e-mail inválido")
	ErrWeakPassword       = errors.New("a senha deve ter pelo menos 10 caracteres")
	ErrInvalidRole        = errors.New("papel inválido: use admin, central ou legal_reviewer")
)
