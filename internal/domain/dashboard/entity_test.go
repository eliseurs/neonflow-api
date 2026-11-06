package dashboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/pkg/timezone"
)

func TestMonthRange(t *testing.T) {
	// 01/12 às 01h UTC ainda é 30/11 em Gramado: o mês atual é novembro.
	now := time.Date(2026, 12, 1, 1, 0, 0, 0, time.UTC)
	m, start, end, err := MonthRange("", now)
	require.NoError(t, err)
	assert.Equal(t, "2026-11", m)
	assert.True(t, start.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, timezone.Location)))
	assert.True(t, end.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, timezone.Location)))

	m, _, end, err = MonthRange("2026-12", now)
	require.NoError(t, err)
	assert.Equal(t, "2026-12", m)
	assert.Equal(t, 2027, end.Year())

	for _, bad := range []string{"2026-13", "2026-1", "dez", "2026-12-01"} {
		_, _, _, err = MonthRange(bad, now)
		assert.ErrorIs(t, err, ErrInvalidMonth, bad)
	}
}

func TestOccupancy(t *testing.T) {
	assert.Equal(t, 0.0, Occupancy(0, 0))
	assert.Equal(t, 50.0, Occupancy(1, 2))
	assert.Equal(t, 33.3, Occupancy(1, 3))
	assert.Equal(t, 100.0, Occupancy(2, 2))
}
