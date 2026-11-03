//go:build integration

package migrator_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/infra/migrator"
	"neonflow-api/internal/testutil"
)

func TestMigrator_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../sql/migrations")
	ctx := context.Background()

	dir := t.TempDir()
	files := map[string]string{
		"V1_a.up.sql":   "CREATE TABLE IF NOT EXISTS mig_a (id int);",
		"V1_a.down.sql": "DROP TABLE IF EXISTS mig_a;",
		"V2_b.up.sql":   "CREATE TABLE IF NOT EXISTS mig_b (id int);",
		"V2_b.down.sql": "DROP TABLE IF EXISTS mig_b;",
	}
	for n, body := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, n), []byte(body), 0o600))
	}
	_, err := database.ExecContext(ctx, `DROP TABLE IF EXISTS schema_migrations`)
	require.NoError(t, err)

	var out bytes.Buffer
	m := migrator.New(database, dir)
	m.SetOutput(&out)

	require.NoError(t, m.Up(ctx))
	assert.Contains(t, out.String(), "V2_b.up.sql")
	out.Reset()
	require.NoError(t, m.Up(ctx))
	assert.Contains(t, out.String(), "nothing to migrate")

	out.Reset()
	require.NoError(t, m.Status(ctx))
	assert.Contains(t, out.String(), "applied")

	require.NoError(t, m.Down(ctx))
	require.NoError(t, m.Down(ctx))
	assert.ErrorIs(t, m.Down(ctx), migrator.ErrNothingToRollback)

	out.Reset()
	require.NoError(t, m.Status(ctx))
	assert.Contains(t, out.String(), "pending")

	// SQL inválido em V3: V1 e V2 aplicam, V3 falha e não é registrada (transação).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "V3_bad.up.sql"), []byte("SELECT * FROM tabela_que_nao_existe;"), 0o600))
	assert.Error(t, m.Up(ctx))
	var applied int
	require.NoError(t, database.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&applied))
	assert.Equal(t, 2, applied)

	// Corrigida, V3 aplica. Sem .down.sql, o rollback dela é recusado.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "V3_bad.up.sql"), []byte("SELECT 1;"), 0o600))
	require.NoError(t, m.Up(ctx))
	assert.ErrorContains(t, m.Down(ctx), "não tem arquivo .down.sql")

	// Diretório inválido e banco fechado.
	assert.Error(t, migrator.New(database, filepath.Join(dir, "nao-existe")).Up(ctx))
	require.NoError(t, database.Close())
	assert.Error(t, m.Up(ctx))
	assert.Error(t, m.Down(ctx))
	assert.Error(t, m.Status(ctx))
}
