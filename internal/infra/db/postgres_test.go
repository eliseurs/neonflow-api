package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpen_Unreachable(t *testing.T) {
	_, err := Open(context.Background(), "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1")
	assert.ErrorContains(t, err, "ping db")
}
