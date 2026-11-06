package main

import (
	"context"
	"os"

	"neonflow-api/internal/config"
	authRoute "neonflow-api/internal/domain/authentication/route"
	bookingRoute "neonflow-api/internal/domain/booking/route"
	checklistRoute "neonflow-api/internal/domain/checklist/route"
	dashboardRoute "neonflow-api/internal/domain/dashboard/route"
	fleetRoute "neonflow-api/internal/domain/fleet/route"
	partnerRoute "neonflow-api/internal/domain/partner/route"
	payoutRoute "neonflow-api/internal/domain/payout/route"
	userRoute "neonflow-api/internal/domain/user/route"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/infra/db"
	"neonflow-api/internal/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.IsDeployed())

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
	fleetRoute.Register(app)
	bookingRoute.Register(app)
	checklistRoute.Register(app)
	userRoute.Register(app)
	partnerRoute.Register(app)
	payoutRoute.Register(app)
	dashboardRoute.Register(app)

	app.Server()
}
