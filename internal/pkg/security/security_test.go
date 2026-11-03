package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenProvider_RoundTrip(t *testing.T) {
	p := NewTokenProvider("segredo", time.Hour)
	token, exp, err := p.Generate("user-1", "central")
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), exp, 5*time.Second)

	claims, err := p.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "central", claims.Role)
}

func TestTokenProvider_Rejects(t *testing.T) {
	p := NewTokenProvider("segredo", time.Hour)
	valid, _, err := p.Generate("user-1", "admin")
	require.NoError(t, err)

	expired := NewTokenProvider("segredo", time.Hour)
	expired.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	old, _, _ := expired.Generate("user-1", "admin")

	none := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: "u", Role: "admin"})
	noneToken, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)

	noRole, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:           "u",
		RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}).SignedString([]byte("segredo"))

	cases := map[string]string{
		"assinatura de outro segredo": func() string { t, _, _ := NewTokenProvider("outro", time.Hour).Generate("u", "admin"); return t }(),
		"expirado":                    old,
		"alg none":                    noneToken,
		"sem papel":                   noRole,
		"lixo":                        "abc.def.ghi",
		"vazio":                       "",
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := p.Parse(token)
			assert.ErrorIs(t, err, ErrInvalidToken)
		})
	}
	_, err = p.Parse(valid)
	assert.NoError(t, err)
}

func TestBcryptHasher(t *testing.T) {
	h := &BcryptHasher{cost: 4}
	hash, err := h.Hash("senha-forte-123")
	require.NoError(t, err)
	assert.NotEqual(t, "senha-forte-123", hash)
	assert.True(t, h.Compare(hash, "senha-forte-123"))
	assert.False(t, h.Compare(hash, "errada"))
	assert.False(t, h.Compare("nao-e-hash", "senha-forte-123"))

	_, err = h.Hash(string(make([]byte, 80)))
	assert.Error(t, err, "bcrypt recusa senha acima de 72 bytes")
	assert.NotNil(t, NewBcryptHasher())
}
