// Package authz é a matriz de acesso do painel (RN-05.10): cada papel tem uma lista de permissões,
// e toda rota protegida exige uma delas. O app recebe a mesma lista em /auth/me e monta o menu por ela.
package authz

import "slices"

const (
	RoleAdmin   = "admin"
	RoleCentral = "central"
	// RoleLegalReviewer é a revisora jurídica dos contratos (RN-08.4). Não acessa a operação.
	RoleLegalReviewer = "legal_reviewer"
	// RolePartner é a conta de um parceiro (proprietário, motorista ou indicador), criada por convite (RN-05.12).
	RolePartner = "parceiro"
	// RoleCustomer é a conta do cliente, criada por ele no site com o e-mail confirmado (RN-05.13).
	RoleCustomer = "cliente"
)

const (
	// Operation é o dia a dia da central: visão geral, reservas, transfer, frota, clientes, parceiros e repasses.
	Operation          = "operacao"
	CustomersAnonymize = "clientes.anonimizar"
	TermsPublish       = "termos.publicar"
	Contracts          = "contratos"
	Users              = "usuarios"
	Audit              = "auditoria"
	// PartnerArea e CustomerArea são as áreas próprias: cada um vê só o que é seu.
	PartnerArea  = "area.parceiro"
	CustomerArea = "area.cliente"
)

// A matriz repete o que as rotas liberavam por papel até 16/11/2026. Mudar o que um papel pode fazer é mudar aqui.
var matrix = map[string][]string{
	RoleAdmin:         {Operation, CustomersAnonymize, TermsPublish, Contracts, Users, Audit},
	RoleCentral:       {Operation},
	RoleLegalReviewer: {Contracts},
	RolePartner:       {PartnerArea},
	RoleCustomer:      {CustomerArea},
}

// StaffRole diz se o papel é da equipe, os que o admin cria e edita em Usuários. Parceiro e cliente
// têm conta ligada ao próprio cadastro e não entram por lá.
func StaffRole(role string) bool {
	return role == RoleAdmin || role == RoleCentral || role == RoleLegalReviewer
}

func Can(role, permission string) bool {
	return slices.Contains(matrix[role], permission)
}

// Permissions devolve as permissões do papel; papel desconhecido não tem nenhuma.
func Permissions(role string) []string {
	return append([]string{}, matrix[role]...)
}
