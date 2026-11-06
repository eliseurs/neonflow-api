package customer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestValidate(t *testing.T) {
	r, err := Request{Name: " Cliente Teste ", WhatsApp: "+55 (54) 99999-0000", Email: " Cliente@Exemplo.TEST ", Notes: " vip "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Cliente Teste", r.Name)
	assert.Equal(t, "5554999990000", r.WhatsApp)
	assert.Equal(t, "cliente@exemplo.test", r.Email)
	assert.Equal(t, LangPT, r.Language, "padrão: português")
	assert.Equal(t, "vip", r.Notes)
	en, err := Request{Name: "Guest", Language: " EN "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, LangEN, en.Language)

	cases := []struct {
		name string
		req  Request
		want error
	}{
		{"nome", Request{Name: " "}, ErrNameRequired},
		{"whatsapp curto", Request{Name: "X", WhatsApp: "999"}, ErrInvalidWhatsApp},
		{"whatsapp longo", Request{Name: "X", WhatsApp: "1234567890123456"}, ErrInvalidWhatsApp},
		{"e-mail", Request{Name: "X", Email: "sem-arroba"}, ErrInvalidEmail},
		{"idioma", Request{Name: "X", Language: "es"}, ErrInvalidLanguage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.req.Validate()
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	text, digits := NormalizeQuery(" João ")
	assert.Equal(t, "joão", text)
	assert.Equal(t, "", digits)
	_, digits = NormalizeQuery("(54) 99999-0000")
	assert.Equal(t, "54999990000", digits)
}
