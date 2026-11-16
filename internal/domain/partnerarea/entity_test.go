package partnerarea

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPartnerHas(t *testing.T) {
	p := Partner{Kinds: []string{KindOwner, "indicador"}}
	assert.True(t, p.Has(KindOwner))
	assert.False(t, p.Has(KindDriver))
}
