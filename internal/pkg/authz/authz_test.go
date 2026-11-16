package authz

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatrix(t *testing.T) {
	assert.Equal(t, []string{Operation, CustomersAnonymize, TermsPublish, Contracts, Users, Audit}, Permissions(RoleAdmin))
	assert.Equal(t, []string{Operation}, Permissions(RoleCentral))
	assert.Equal(t, []string{Contracts}, Permissions(RoleLegalReviewer), "a revisora só vê contratos (RN-08.4)")
	assert.Equal(t, []string{PartnerArea}, Permissions(RolePartner))
	assert.Equal(t, []string{CustomerArea}, Permissions(RoleCustomer))
	assert.Equal(t, []string{}, Permissions("motorista"), "papel desconhecido não tem permissão")

	assert.True(t, Can(RoleAdmin, Users))
	assert.False(t, Can(RoleCentral, Users))
	assert.False(t, Can(RoleCentral, Contracts), "a central não vê os contratos (RN-08.4)")
	assert.False(t, Can(RoleLegalReviewer, Operation))
	assert.False(t, Can("", Operation))
	assert.False(t, Can(RolePartner, Operation), "o parceiro não entra na operação da central")
	assert.False(t, Can(RoleCustomer, Operation), "o cliente não entra na operação da central")
	assert.False(t, Can(RoleCustomer, PartnerArea))
}

func TestStaffRole(t *testing.T) {
	assert.True(t, StaffRole(RoleAdmin))
	assert.True(t, StaffRole(RoleCentral))
	assert.True(t, StaffRole(RoleLegalReviewer))
	assert.False(t, StaffRole(RolePartner), "conta de parceiro só por convite")
	assert.False(t, StaffRole(RoleCustomer), "conta de cliente só pelo cadastro no site")
	assert.False(t, StaffRole("motorista"))
}

func TestPermissions_Copy(t *testing.T) {
	p := Permissions(RoleCentral)
	p[0] = Users
	assert.False(t, Can(RoleCentral, Users), "quem recebe a lista não altera a matriz")
}
