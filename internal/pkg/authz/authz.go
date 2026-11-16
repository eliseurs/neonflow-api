// Package authz é a matriz de acesso do painel (RN-05.10): cada papel tem uma lista de permissões,
// e toda rota protegida exige uma delas. O app recebe a mesma lista em /auth/me e monta o menu por ela.
package authz

import "slices"

const (
	RoleAdmin   = "admin"
	RoleCentral = "central"
	// RoleLegalReviewer é a revisora jurídica dos contratos (RN-08.4). Não acessa a operação.
	RoleLegalReviewer = "legal_reviewer"
)

const (
	// Operation é o dia a dia da central: visão geral, reservas, transfer, frota, clientes, parceiros e repasses.
	Operation          = "operacao"
	CustomersAnonymize = "clientes.anonimizar"
	TermsPublish       = "termos.publicar"
	Contracts          = "contratos"
	Users              = "usuarios"
	Audit              = "auditoria"
)

// A matriz repete o que as rotas liberavam por papel até 16/11/2026. Mudar o que um papel pode fazer é mudar aqui.
var matrix = map[string][]string{
	RoleAdmin:         {Operation, CustomersAnonymize, TermsPublish, Contracts, Users, Audit},
	RoleCentral:       {Operation},
	RoleLegalReviewer: {Contracts},
}

func ValidRole(role string) bool {
	_, ok := matrix[role]
	return ok
}

func Can(role, permission string) bool {
	return slices.Contains(matrix[role], permission)
}

// Permissions devolve as permissões do papel; papel desconhecido não tem nenhuma.
func Permissions(role string) []string {
	return append([]string{}, matrix[role]...)
}
