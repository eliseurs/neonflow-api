package account

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/authentication"
)

func TestSignupRequest_Validate(t *testing.T) {
	ok := SignupRequest{Name: "  Ana Teste ", Email: " ANA@neonflow.test ", Password: "senha-forte-1"}
	got, err := ok.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Ana Teste", got.Name)
	assert.Equal(t, "ana@neonflow.test", got.Email)

	cases := map[string]struct {
		req  SignupRequest
		want error
	}{
		"sem nome":     {SignupRequest{Name: " ", Email: "a@neonflow.test", Password: "senha-forte-1"}, authentication.ErrNameRequired},
		"e-mail ruim":  {SignupRequest{Name: "Ana", Email: "Ana <a@neonflow.test>", Password: "senha-forte-1"}, authentication.ErrInvalidEmail},
		"senha curta":  {SignupRequest{Name: "Ana", Email: "a@neonflow.test", Password: "curta"}, authentication.ErrWeakPassword},
		"e-mail vazio": {SignupRequest{Name: "Ana", Password: "senha-forte-1"}, authentication.ErrInvalidEmail},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := tc.req.Validate()
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestNewToken(t *testing.T) {
	a, hashA, err := NewToken()
	require.NoError(t, err)
	b, _, err := NewToken()
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	assert.Len(t, a, 43, "32 bytes em base64 sem preenchimento")
	assert.Equal(t, hashA, HashToken(a))
	assert.Equal(t, hashA, HashToken(" "+a+" "), "espaço colado junto com o link não muda o código")
	assert.NotContains(t, hashA, a, "o banco guarda só o hash")
}
