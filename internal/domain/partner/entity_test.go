package partner

import (
	"math"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pct(v float64) *float64 { return &v }

func TestRequestValidate(t *testing.T) {
	r, err := Request{Name: " Hotel Teste ", Kinds: []string{" Indicador ", "proprietario", "indicador"},
		WhatsApp: "+55 (54) 99999-0000", Email: " Contato@Hotel.test ", CommissionPercent: pct(10),
		OwnerSharePercent: pct(70), DriverSharePercent: pct(80)}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Hotel Teste", r.Name)
	assert.Equal(t, []string{KindReferrer, KindOwner}, r.Kinds, "sem repetição")
	assert.Equal(t, "5554999990000", r.WhatsApp)
	assert.Equal(t, "contato@hotel.test", r.Email)
	assert.Equal(t, 70.0, *r.OwnerSharePercent)
	assert.Nil(t, r.DriverSharePercent, "percentual de motorista só para motorista")

	cases := []struct {
		name string
		req  Request
		want error
	}{
		{"sem nome", Request{Name: " ", Kinds: []string{KindReferrer}}, ErrNameRequired},
		{"sem tipo", Request{Name: "X"}, ErrInvalidKind},
		{"tipo inválido", Request{Name: "X", Kinds: []string{"locadora"}}, ErrInvalidKind},
		{"e-mail", Request{Name: "X", Kinds: []string{KindDriver}, Email: "sem-arroba"}, ErrInvalidEmail},
		{"percentual alto", Request{Name: "X", Kinds: []string{KindReferrer}, CommissionPercent: pct(101)}, ErrInvalidPercent},
		{"percentual negativo", Request{Name: "X", Kinds: []string{KindDriver}, DriverSharePercent: pct(-1)}, ErrInvalidPercent},
		{"NaN", Request{Name: "X", Kinds: []string{KindReferrer}, CommissionPercent: pct(math.NaN())}, ErrInvalidPercent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.req.Validate()
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestBasisPoints(t *testing.T) {
	assert.Nil(t, ToBasisPoints(nil))
	assert.Nil(t, FromBasisPoints(nil))
	assert.Equal(t, 1250, *ToBasisPoints(pct(12.5)))
	bp := 333
	assert.Equal(t, 3.33, *FromBasisPoints(&bp))
}

func TestReferralCode(t *testing.T) {
	code, err := NewReferralCode()
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^[A-HJ-NP-Z2-9]{6}$`), code, "sem 0, O, 1, I ou L")
	assert.Equal(t, "ABC234", NormalizeReferralCode(" abc234 "))
	assert.Equal(t, "", NormalizeReferralCode("ab"))
	assert.Equal(t, "", NormalizeReferralCode("abc-234"))
}
