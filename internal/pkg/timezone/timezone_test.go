package timezone

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocation(t *testing.T) {
	assert.Equal(t, "America/Sao_Paulo", Location.String())
	// 02:30 UTC ainda é o dia anterior em Gramado (UTC-3).
	assert.Equal(t, "2026-12-09", Date(time.Date(2026, 12, 10, 2, 30, 0, 0, time.UTC)))
	assert.Equal(t, "2026-12-10", Date(time.Date(2026, 12, 10, 3, 0, 0, 0, time.UTC)))
}
