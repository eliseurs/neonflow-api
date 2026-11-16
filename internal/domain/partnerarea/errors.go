package partnerarea

import "errors"

// ErrNoPartner: a conta logada não é de um parceiro ativo.
var ErrNoPartner = errors.New("entre com a sua conta de parceiro")
