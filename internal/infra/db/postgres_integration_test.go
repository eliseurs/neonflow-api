//go:build integration

package db

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpen_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	database, err := Open(context.Background(), dsn)
	require.NoError(t, err)
	require.NoError(t, database.Close())
}
