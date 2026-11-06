package payout

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAmount(t *testing.T) {
	assert.Equal(t, int64(3000), Amount(30000, 1000), "10% de R$ 300")
	assert.Equal(t, int64(3750), Amount(30000, 1250), "12,5%")
	assert.Equal(t, int64(33), Amount(333, 1000), "arredonda")
	assert.Equal(t, int64(0), Amount(30000, 0))
}

func TestFilterValidate(t *testing.T) {
	assert.NoError(t, Filter{}.Validate())
	assert.NoError(t, Filter{Kind: KindOwnerShare, Status: StatusPaid}.Validate())
	assert.ErrorIs(t, Filter{Kind: "bonus"}.Validate(), ErrInvalidFilter)
	assert.ErrorIs(t, Filter{Status: "atrasada"}.Validate(), ErrInvalidFilter)
}
