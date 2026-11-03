package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// Migrator aplica migrations versionadas de um diretório.
//
// Convenção de nomes:
//
//	sql/migrations/V{N}_descricao.up.sql
//	sql/migrations/V{N}_descricao.down.sql
//
// Versões aplicadas ficam na tabela schema_migrations ("V1", "V2", ...).
type Migrator struct {
	db  *sql.DB
	dir string
	out io.Writer
}

var fileRe = regexp.MustCompile(`^V(\d+)_[a-z0-9_]+\.(up|down)\.sql$`)

var ErrNothingToRollback = errors.New("nenhuma migration aplicada para desfazer")

func New(db *sql.DB, dir string) *Migrator {
	return &Migrator{db: db, dir: dir, out: os.Stdout}
}

// SetOutput troca o destino do progresso. nil descarta (útil em testes).
func (m *Migrator) SetOutput(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	m.out = w
}

// Up aplica, em ordem, cada migration pendente dentro de uma transação.
func (m *Migrator) Up(ctx context.Context) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}
	done, err := m.applied(ctx)
	if err != nil {
		return err
	}
	versions, err := Versions(m.dir)
	if err != nil {
		return err
	}

	count := 0
	for _, v := range versions {
		if done[v.Name()] {
			continue
		}
		fmt.Fprintf(m.out, "  applying  %-44s ", filepath.Base(v.Up))
		if err := m.exec(ctx, v.Up, `INSERT INTO schema_migrations (version) VALUES ($1)`, v.Name()); err != nil {
			fmt.Fprintln(m.out, "FAIL")
			return fmt.Errorf("migration %s: %w", v.Name(), err)
		}
		fmt.Fprintln(m.out, "ok")
		count++
	}
	if count == 0 {
		fmt.Fprintln(m.out, "  nothing to migrate — schema is up to date")
	}
	return nil
}

// Down desfaz a última migration aplicada.
func (m *Migrator) Down(ctx context.Context) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}
	done, err := m.applied(ctx)
	if err != nil {
		return err
	}
	versions, err := Versions(m.dir)
	if err != nil {
		return err
	}
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		if !done[v.Name()] {
			continue
		}
		if v.Down == "" {
			return fmt.Errorf("migration %s não tem arquivo .down.sql", v.Name())
		}
		fmt.Fprintf(m.out, "  reverting %-44s ", filepath.Base(v.Down))
		if err := m.exec(ctx, v.Down, `DELETE FROM schema_migrations WHERE version = $1`, v.Name()); err != nil {
			fmt.Fprintln(m.out, "FAIL")
			return fmt.Errorf("rollback %s: %w", v.Name(), err)
		}
		fmt.Fprintln(m.out, "ok")
		return nil
	}
	return ErrNothingToRollback
}

// Status lista cada migration como applied ou pending.
func (m *Migrator) Status(ctx context.Context) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}
	done, err := m.applied(ctx)
	if err != nil {
		return err
	}
	versions, err := Versions(m.dir)
	if err != nil {
		return err
	}
	for _, v := range versions {
		state := "pending"
		if done[v.Name()] {
			state = "applied"
		}
		fmt.Fprintf(m.out, "  %-8s %s\n", state, filepath.Base(v.Up))
	}
	return nil
}

func (m *Migrator) ensureTable(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

func (m *Migrator) applied(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("list applied: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	done := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		done[v] = true
	}
	return done, rows.Err()
}

func (m *Migrator) exec(ctx context.Context, path, record, version string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, record, version); err != nil {
		return err
	}
	return tx.Commit()
}

type Version struct {
	Number int
	Up     string
	Down   string
}

func (v Version) Name() string { return "V" + strconv.Itoa(v.Number) }

// Versions lê o diretório e devolve as migrations em ordem, validando nomes e sequência.
func Versions(dir string) ([]Version, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	byNum := map[int]*Version{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		match := fileRe.FindStringSubmatch(e.Name())
		if match == nil {
			return nil, fmt.Errorf("nome de migration inválido: %s (use V{N}_descricao.up.sql)", e.Name())
		}
		n, _ := strconv.Atoi(match[1])
		v, ok := byNum[n]
		if !ok {
			v = &Version{Number: n}
			byNum[n] = v
		}
		path := filepath.Join(dir, e.Name())
		if match[2] == "up" {
			if v.Up != "" {
				return nil, fmt.Errorf("versão V%d duplicada", n)
			}
			v.Up = path
		} else {
			if v.Down != "" {
				return nil, fmt.Errorf("versão V%d duplicada", n)
			}
			v.Down = path
		}
	}

	out := make([]Version, 0, len(byNum))
	for _, v := range byNum {
		if v.Up == "" {
			return nil, fmt.Errorf("V%d sem arquivo .up.sql", v.Number)
		}
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	for i, v := range out {
		if v.Number != i+1 {
			return nil, fmt.Errorf("sequência de migrations quebrada: esperado V%d, encontrado V%d", i+1, v.Number)
		}
	}
	return out, nil
}
