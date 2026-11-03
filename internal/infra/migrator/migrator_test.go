package migrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func write(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		require.NoError(t, os.WriteFile(filepath.Join(dir, n), []byte("SELECT 1;"), 0o600))
	}
}

func TestVersions_OrderedAndPaired(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "V2_b.up.sql", "V2_b.down.sql", "V10_j.up.sql", "V1_a.up.sql", "V1_a.down.sql")
	for i := 3; i <= 9; i++ {
		write(t, dir, "V"+string(rune('0'+i))+"_x.up.sql")
	}
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o700))

	vs, err := Versions(dir)
	require.NoError(t, err)
	require.Len(t, vs, 10)
	assert.Equal(t, "V1", vs[0].Name())
	assert.Equal(t, "V10", vs[9].Name(), "ordena numericamente, não por texto")
	assert.NotEmpty(t, vs[0].Down)
	assert.Empty(t, vs[9].Down)
}

func TestVersions_Errors(t *testing.T) {
	cases := map[string][]string{
		"nome inválido":      {"V1_Upper.up.sql"},
		"sem up":             {"V1_a.down.sql"},
		"sequência quebrada": {"V1_a.up.sql", "V3_c.up.sql"},
		"up duplicado":       {"V1_a.up.sql", "V1_b.up.sql"},
		"down duplicado":     {"V1_a.up.sql", "V1_a.down.sql", "V1_b.down.sql"},
	}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, files...)
			_, err := Versions(dir)
			assert.Error(t, err)
		})
	}
	_, err := Versions(filepath.Join(t.TempDir(), "nao-existe"))
	assert.Error(t, err)
}

func TestVersions_RepoMigrations(t *testing.T) {
	vs, err := Versions("../../../sql/migrations")
	require.NoError(t, err, "as migrations do repositório seguem a convenção")
	assert.NotEmpty(t, vs)
	for _, v := range vs {
		assert.NotEmpty(t, v.Down, "%s precisa de .down.sql", v.Name())
	}
}
