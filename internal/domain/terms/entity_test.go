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

func TestAcceptRequestValidate_Order(t *testing.T) {
	ok := AcceptRequest{TypedName: " Ana  Teste ", ContractVersionID: " cv ", SHA256: " AB ", QuadroSHA256: " CD ", Confirm: true}
	r, err := ok.Validate()
	require.NoError(t, err)
	assert.True(t, r.IsOrder())
	assert.Equal(t, "cv", r.ContractVersionID)
	assert.Equal(t, "ab", r.SHA256)
	assert.Equal(t, "cd", r.QuadroSHA256)

	noConfirm := ok
	noConfirm.Confirm = false
	_, err = noConfirm.Validate()
	assert.ErrorIs(t, err, ErrConfirmRequired)
	noHash := ok
	noHash.QuadroSHA256 = ""
	_, err = noHash.Validate()
	assert.ErrorIs(t, err, ErrVersionChanged)
}

func TestQuadroHash(t *testing.T) {
	km := 200
	q := Quadro{ClientName: "Ana Teste", TotalCents: 100, KmLimitPerDay: &km}
	h1, b, err := q.Hash()
	require.NoError(t, err)
	assert.Len(t, h1, 64)
	assert.Contains(t, string(b), `"km_limit_per_day":200`)
	q.TotalCents = 101
	h2, _, _ := q.Hash()
	assert.NotEqual(t, h1, h2, "mudar o quadro muda o hash")
}
