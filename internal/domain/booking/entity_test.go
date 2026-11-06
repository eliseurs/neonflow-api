package booking

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePeriod(t *testing.T) {
	p, err := ParsePeriod(" 2026-12-10 ", "2026-12-13")
	require.NoError(t, err)
	assert.Equal(t, 3, p.Days())

	cases := []struct {
		name, start, end string
		want             error
	}{
		{"30/02 não existe (RN-02.7)", "2026-02-30", "2026-03-02", ErrInvalidDate},
		{"formato brasileiro", "10/12/2026", "2026-12-13", ErrInvalidDate},
		{"fim inválido", "2026-12-10", "", ErrInvalidDate},
		{"mesmo dia (RN-02.12)", "2026-12-10", "2026-12-10", ErrInvalidPeriod},
		{"fim antes do início", "2026-12-10", "2026-12-09", ErrInvalidPeriod},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParsePeriod(tc.start, tc.end)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestCreateRequestValidate(t *testing.T) {
	req := CreateRequest{VehicleID: " v1 ", ClientName: " Cliente ", StartDate: "2026-12-10", EndDate: "2026-12-12", Place: " Gramado "}
	n, err := req.Validate("u1")
	require.NoError(t, err)
	assert.Equal(t, "v1", n.VehicleID)
	assert.Equal(t, "Cliente", n.ClientName)
	assert.Equal(t, "Gramado", n.Place)
	assert.Equal(t, "u1", n.CreatedBy)
	assert.Equal(t, 2, n.Period.Days())

	_, err = CreateRequest{ClientName: "C", StartDate: "2026-12-10", EndDate: "2026-12-12"}.Validate("u1")
	assert.ErrorIs(t, err, ErrVehicleRequired)
	_, err = CreateRequest{VehicleID: "v1", StartDate: "2026-12-10", EndDate: "2026-12-12"}.Validate("u1")
	assert.ErrorIs(t, err, ErrClientNameRequired)
	_, err = CreateRequest{VehicleID: "v1", ClientName: "C", StartDate: "2026-02-30", EndDate: "2026-12-12"}.Validate("u1")
	assert.ErrorIs(t, err, ErrInvalidDate)
}

func TestValidateChecks(t *testing.T) {
	assert.NoError(t, ValidateChecks(StagePreparacao, map[string]bool{"cnh": true, "payment": false}))
	assert.NoError(t, ValidateChecks(StageConfirmada, map[string]bool{"delivery": true}))
	assert.ErrorIs(t, ValidateChecks(StagePreparacao, map[string]bool{"delivery": true}), ErrInvalidCheck)
	assert.ErrorIs(t, ValidateChecks(StagePreparacao, map[string]bool{}), ErrInvalidCheck)
	assert.ErrorIs(t, ValidateChecks(StageEncerrada, map[string]bool{"issues": true}), ErrStageFinal)
	assert.ErrorIs(t, ValidateChecks(StageCancelada, map[string]bool{"cnh": true}), ErrStageFinal)
}

func TestNextStage(t *testing.T) {
	all := map[string]bool{"cnh": true, "terms": true, "deposit": true, "payment": true,
		"delivery": true, "return": true, "issues": true}
	flow := []string{StagePreparacao, StageConfirmada, StageEmLocacao, StageDevolvida, StageEncerrada}
	for i := 0; i < len(flow)-1; i++ {
		next, err := NextStage(flow[i], all)
		require.NoError(t, err)
		assert.Equal(t, flow[i+1], next)
	}

	_, err := NextStage(StagePreparacao, map[string]bool{"cnh": true, "terms": true, "deposit": true})
	assert.ErrorIs(t, err, ErrChecksPending)
	assert.Contains(t, err.Error(), "payment", "RN-02.3: sem pagamento não confirma")

	_, err = NextStage(StageEncerrada, all)
	assert.ErrorIs(t, err, ErrStageFinal, "RN-02.4: encerrada não avança")
	_, err = NextStage(StageCancelada, all)
	assert.ErrorIs(t, err, ErrStageFinal)
}

func TestChecklistExtras(t *testing.T) {
	// RN-01.6: em "devolvida", os itens do checklist entram nas conferências exigidas.
	items := []string{"combustivel", "pneus"}
	assert.NoError(t, ValidateChecks(StageDevolvida, map[string]bool{"issues": true, "pneus": true}, items...))
	assert.ErrorIs(t, ValidateChecks(StageDevolvida, map[string]bool{"oleo": true}, items...), ErrInvalidCheck)

	_, err := NextStage(StageDevolvida, map[string]bool{"issues": true, "combustivel": true}, items...)
	assert.ErrorIs(t, err, ErrChecksPending)
	assert.Contains(t, err.Error(), "pneus")
	next, err := NextStage(StageDevolvida, map[string]bool{"issues": true, "combustivel": true, "pneus": true}, items...)
	require.NoError(t, err)
	assert.Equal(t, StageEncerrada, next)

	assert.ElementsMatch(t, []string{"cnh", "terms", "deposit", "payment", "delivery", "return", "issues"}, ReservedCheckKeys())
}

func TestCancelable(t *testing.T) {
	assert.True(t, Cancelable(StagePreparacao))
	assert.True(t, Cancelable(StageConfirmada))
	for _, s := range []string{StageEmLocacao, StageDevolvida, StageEncerrada, StageCancelada} {
		assert.False(t, Cancelable(s), s)
	}
}
