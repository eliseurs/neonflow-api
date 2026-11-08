package authentication

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateUserRequest_Validate(t *testing.T) {
	ok := CreateUserRequest{Name: "  Central  ", Email: " Central@NeonFlow.test ", Password: strings.Repeat("a", MinPasswordLength), Role: "central"}
	got, err := ok.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "Central", got.Name)
	assert.Equal(t, "central@neonflow.test", got.Email)

	cases := map[string]struct {
		mutate func(*CreateUserRequest)
		want   error
	}{
		"nome vazio":   {func(r *CreateUserRequest) { r.Name = " " }, ErrNameRequired},
		"email ruim":   {func(r *CreateUserRequest) { r.Email = "sem-arroba" }, ErrInvalidEmail},
		"email nome":   {func(r *CreateUserRequest) { r.Email = "Fulano <f@x.test>" }, ErrInvalidEmail},
		"senha curta":  {func(r *CreateUserRequest) { r.Password = "curta" }, ErrWeakPassword},
		"papel errado": {func(r *CreateUserRequest) { r.Role = "cliente" }, ErrInvalidRole},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := ok
			tc.mutate(&r)
			_, err := r.Validate()
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestValidRole(t *testing.T) {
	assert.True(t, ValidRole(RoleAdmin))
	assert.True(t, ValidRole(RoleCentral))
	assert.True(t, ValidRole(RoleLegalReviewer))
	assert.False(t, ValidRole("motorista"))
}
