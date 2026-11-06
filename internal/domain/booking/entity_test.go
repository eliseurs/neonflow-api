package booking

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/pkg/timezone"
)

func TestParsePeriod(t *testing.T) {
	// Só a data continua aceita e vira 09:00 de Gramado: 3 diárias de 24 horas.
	p, err := ParsePeriod(" 2026-12-10 ", "2026-12-13")
	require.NoError(t, err)
	assert.Equal(t, 3, p.Days())
	assert.Equal(t, time.Date(2026, 12, 10, 9, 0, 0, 0, timezone.Location), p.Start)

	cases := []struct {
		name, start, end string
		want             error
	}{
		{"30/02 não existe (RN-02.7)", "2026-02-30", "2026-03-02", ErrInvalidDate},
		{"formato brasileiro", "10/12/2026", "2026-12-13", ErrInvalidDate},
		{"hora inválida", "2026-12-10T25:00", "2026-12-13T10:00", ErrInvalidDate},
		{"fim inválido", "2026-12-10", "", ErrInvalidDate},
		{"mesmo instante", "2026-12-10T10:00", "2026-12-10T10:00", ErrInvalidPeriod},
		{"fim antes do início", "2026-12-10T10:00", "2026-12-10T09:59", ErrInvalidPeriod},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParsePeriod(tc.start, tc.end)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestParseInstant(t *testing.T) {
	local := time.Date(2026, 12, 10, 14, 30, 0, 0, timezone.Location)
	for _, v := range []string{"2026-12-10T14:30", "2026-12-10T14:30:00", "2026-12-10T14:30:00-03:00", "2026-12-10T17:30:00Z"} {
		got, err := ParseInstant(v)
		require.NoError(t, err, v)
		assert.True(t, local.Equal(got), v)
	}
}

func TestPeriodDays(t *testing.T) {
	// RN-02.24: blocos de 24 horas a partir da retirada, arredondando para cima; mínimo 1 (RN-02.12).
	cases := []struct {
		start, end string
		days       int
	}{
		{"2026-12-10T14:00", "2026-12-11T14:00", 1},
		{"2026-12-10T14:00", "2026-12-11T15:00", 2},
		{"2026-12-10T14:00", "2026-12-12T13:59", 2},
		{"2026-12-10T10:00", "2026-12-10T13:00", 1},
		{"2026-12-10T22:00", "2026-12-11T08:00", 1},
	}
	for _, tc := range cases {
		p, err := ParsePeriod(tc.start, tc.end)
		require.NoError(t, err)
		assert.Equal(t, tc.days, p.Days(), tc.start+" → "+tc.end)
	}
}

func TestLateMinutes(t *testing.T) {
	end := time.Date(2026, 12, 12, 10, 0, 0, 0, timezone.Location)
	// RN-02.19: até 1 hora de tolerância; passado isso, os minutos exatos desde o combinado.
	cases := []struct {
		returned time.Duration
		minutes  int
	}{
		{-3 * time.Hour, 0},
		{0, 0},
		{time.Hour, 0},
		{time.Hour + time.Second, 60},
		{time.Hour + 5*time.Minute, 65},
		{5*time.Hour + 12*time.Minute + 59*time.Second, 312},
		{30 * time.Hour, 1800},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.minutes, LateMinutes(end, end.Add(tc.returned)), tc.returned.String())
	}
}

func TestLateCents(t *testing.T) {
	// Diária ÷ 1440 por minuto; 1440 minutos = 1 diária.
	assert.Equal(t, int64(0), LateCents(0, 14400))
	assert.Equal(t, int64(650), LateCents(65, 14400))
	assert.Equal(t, int64(14400), LateCents(1440, 14400))
	assert.Equal(t, int64(18000), LateCents(1800, 14400), "30 horas = 1 diária + 6 horas")
	assert.Equal(t, int64(451), LateCents(65, 10000), "arredonda ao centavo")
}

func TestExcessKm(t *testing.T) {
	// RN-02.18: franquia = limite por dia × diárias contratadas (q90).
	assert.Equal(t, 0, ExcessKm(200, 3, 600))
	assert.Equal(t, 0, ExcessKm(200, 3, 120))
	assert.Equal(t, 50, ExcessKm(200, 3, 650))
	assert.Equal(t, 0, ExcessKm(200, 4, 650))
}

func TestCreateRequestValidate(t *testing.T) {
	plan, err := CreateRequest{VehicleID: "v1", ClientName: "C", StartDate: "2026-12-10", EndDate: "2026-12-12", KmPlan: " Livre "}.Validate("u1")
	require.NoError(t, err)
	assert.Equal(t, KmPlanLivre, plan.KmPlan)
	_, err = CreateRequest{VehicleID: "v1", ClientName: "C", StartDate: "2026-12-10", EndDate: "2026-12-12", KmPlan: "ilimitado"}.Validate("u1")
	assert.ErrorIs(t, err, ErrInvalidKmPlan)
	_, err = CreateRequest{VehicleID: "v1", ClientName: "C", StartDate: "2026-12-10", EndDate: "2026-12-12", ReferredByPartnerID: "abc"}.Validate("u1")
	assert.ErrorIs(t, err, ErrReferrerNotFound)
	ref, err := CreateRequest{VehicleID: "v1", ClientName: "C", StartDate: "2026-12-10", EndDate: "2026-12-12",
		ReferredByPartnerID: " 00000000-0000-0000-0000-000000000001 "}.Validate("u1")
	require.NoError(t, err)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", ref.ReferredBy)

	withTime, err := CreateRequest{VehicleID: "v1", ClientName: "C", StartAt: "2026-12-10T14:00", EndAt: "2026-12-12T15:00",
		StartDate: "2026-01-01", EndDate: "2026-01-02"}.Validate("u1")
	require.NoError(t, err)
	assert.Equal(t, 3, withTime.Period.Days(), "start_at e end_at têm prioridade sobre as datas")
	assert.Equal(t, 14, withTime.Period.Start.Hour())

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
