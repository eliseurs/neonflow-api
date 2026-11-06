package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/authentication"
)

func TestUpdateRequestValidate(t *testing.T) {
	r, err := UpdateRequest{Name: " Maria ", Role: " central "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Maria", r.Name)
	assert.Equal(t, "central", r.Role)
	_, err = UpdateRequest{Name: " ", Role: "central"}.Validate()
	assert.ErrorIs(t, err, authentication.ErrNameRequired)
	_, err = UpdateRequest{Name: "Maria", Role: "motorista"}.Validate()
	assert.ErrorIs(t, err, authentication.ErrInvalidRole)
}

func TestValidPassword(t *testing.T) {
	assert.ErrorIs(t, ValidPassword("curta"), authentication.ErrWeakPassword)
	assert.NoError(t, ValidPassword("senha-forte-123"))
}
