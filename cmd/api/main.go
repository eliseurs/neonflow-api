package main

import (
	"context"
	"os"

	"neonflow-api/internal/config"
	authRoute "neonflow-api/internal/domain/authentication/route"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/infra/db"
	"neonflow-api/internal/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.IsProduction())

	if err := cfg.Validate(); err != nil {
		log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	database, err := db.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer database.Close() //nolint:errcheck

	app := api.New(cfg, database, log)
	app.Setup()
	app.HealthCheck()

	authRoute.Register(app)

	app.Server()
}
