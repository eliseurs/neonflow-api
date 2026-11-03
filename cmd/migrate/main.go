package main

import (
	"context"
	"fmt"
	"os"

	"neonflow-api/internal/config"
	"neonflow-api/internal/infra/db"
	"neonflow-api/internal/infra/migrator"
)

const usage = "uso: migrate [up|down|status]"

func main() {
	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}
	if err := run(direction); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func run(direction string) error {
	cfg := config.Load()
	ctx := context.Background()

	database, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer database.Close() //nolint:errcheck

	m := migrator.New(database, migrationsDir())
	switch direction {
	case "up":
		fmt.Println("Running migrations...")
		return m.Up(ctx)
	case "down":
		fmt.Println("Rolling back last migration...")
		return m.Down(ctx)
	case "status":
		fmt.Println("Migration status:")
		return m.Status(ctx)
	default:
		return fmt.Errorf("direção desconhecida %q — %s", direction, usage)
	}
}

func migrationsDir() string {
	if dir := os.Getenv("MIGRATIONS_DIR"); dir != "" {
		return dir
	}
	return "sql/migrations"
}
