package maintenance

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func TestRequestValidate(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	r, err := Request{Date: " 2026-11-06 ", Kind: " Preventiva ", Description: " Troca de óleo ", Workshop: " Oficina ",
		CostCents: ptr(int64(30000)), OdometerKm: ptr(10000)}.Validate(now)
	require.NoError(t, err)
	assert.Equal(t, "preventiva", r.Kind)
	assert.Equal(t, "Troca de óleo", r.Description)
	assert.Equal(t, "Oficina", r.Workshop)

	cases := []struct {
		name string
		req  Request
		want error
	}{
		{"data", Request{Date: "06/11/2026", Kind: "outros", Description: "x"}, ErrInvalidDate},
		{"futuro", Request{Date: "2026-11-07", Kind: "outros", Description: "x"}, ErrFutureDate},
		{"tipo", Request{Date: "2026-11-01", Kind: "lavagem", Description: "x"}, ErrInvalidKind},
		{"descrição", Request{Date: "2026-11-01", Kind: "outros", Description: " "}, ErrDescriptionRequired},
		{"descrição longa", Request{Date: "2026-11-01", Kind: "outros", Description: strings.Repeat("a", 1001)}, ErrDescriptionTooLong},
		{"custo", Request{Date: "2026-11-01", Kind: "outros", Description: "x", CostCents: ptr(int64(-1))}, ErrInvalidCost},
		{"km", Request{Date: "2026-11-01", Kind: "outros", Description: "x", OdometerKm: ptr(-1)}, ErrInvalidOdometer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.req.Validate(now)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}
