package checklist

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyFromLabel(t *testing.T) {
	cases := map[string]string{
		"Combustível":                      "combustivel",
		"  Estética (riscos e amassados) ": "estetica_riscos_e_amassados",
		"Nível de óleo":                    "nivel_de_oleo",
		"Documentos no carro!":             "documentos_no_carro",
		"Calibração — 4 pneus":             "calibracao_4_pneus",
		"???":                              "",
	}
	for label, want := range cases {
		assert.Equal(t, want, KeyFromLabel(label), label)
	}
	long := KeyFromLabel(strings.Repeat("palavra ", 10))
	assert.LessOrEqual(t, len(long), maxKeyLength)
	assert.False(t, strings.HasSuffix(long, "_"))
}

func TestItemRequestValidate(t *testing.T) {
	r, err := ItemRequest{Label: "  Pneus  "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Pneus", r.Label)
	_, err = ItemRequest{Label: "   "}.Validate()
	assert.ErrorIs(t, err, ErrLabelRequired)
}

func TestNewKey(t *testing.T) {
	key, err := NewKey("Calota")
	require.NoError(t, err)
	assert.Equal(t, "calota", key)
	_, err = NewKey("!!!")
	assert.ErrorIs(t, err, ErrInvalidLabel)
	// "Issues" colidiria com a conferência fixa de ocorrências da reserva.
	_, err = NewKey("Issues")
	assert.ErrorIs(t, err, ErrKeyReserved)
}
