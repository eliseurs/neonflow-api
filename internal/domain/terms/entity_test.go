package terms

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptRequestValidate(t *testing.T) {
	r, err := AcceptRequest{TypedName: "  Ana   Maria  Teste ", Version: 2}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Ana Maria Teste", r.TypedName)
	_, err = AcceptRequest{TypedName: "Al", Version: 1}.Validate()
	assert.ErrorIs(t, err, ErrInvalidName)
	_, err = AcceptRequest{TypedName: strings.Repeat("a", 201), Version: 1}.Validate()
	assert.ErrorIs(t, err, ErrInvalidName)
	_, err = AcceptRequest{TypedName: "Ana Teste"}.Validate()
	assert.ErrorIs(t, err, ErrVersionChanged)
}

func TestValidateBody(t *testing.T) {
	b, err := ValidateBody("  Termos do aluguel.  ")
	require.NoError(t, err)
	assert.Equal(t, "Termos do aluguel.", b)
	_, err = ValidateBody("  ")
	assert.ErrorIs(t, err, ErrBodyRequired)
	_, err = ValidateBody(strings.Repeat("a", maxBody+1))
	assert.ErrorIs(t, err, ErrBodyTooLong)
}

func TestTokenAndStage(t *testing.T) {
	token, hash, err := NewToken()
	require.NoError(t, err)
	assert.Len(t, token, 43, "32 bytes em base64 sem preenchimento")
	assert.Equal(t, HashToken(token), hash)
	assert.Len(t, hash, 64)
	other, _, _ := NewToken()
	assert.NotEqual(t, token, other)

	assert.True(t, LinkAllowed("preparacao"))
	assert.True(t, LinkAllowed("confirmada"))
	assert.False(t, LinkAllowed("em_locacao"))
	assert.False(t, LinkAllowed("cancelada"))
}
