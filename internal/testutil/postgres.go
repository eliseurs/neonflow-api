//go:build integration

package testutil

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"neonflow-api/internal/infra/db"
	"neonflow-api/internal/infra/migrator"
)

// MigratedDB abre TEST_DATABASE_URL, recria o schema public e aplica as migrations.
// Nunca aponte TEST_DATABASE_URL para um banco com dados reais.
func MigratedDB(t *testing.T, migrationsDir string) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	ctx := context.Background()
	database, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.ExecContext(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	m := migrator.New(database, migrationsDir)
	m.SetOutput(nil)
	if err := m.Up(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}
