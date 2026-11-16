package route

import (
	"neonflow-api/internal/domain/maintenance/handler"
	"neonflow-api/internal/domain/maintenance/repository"
	"neonflow-api/internal/domain/maintenance/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/authz"
)

// Register expõe o histórico de manutenção dos veículos para a central.
func Register(app *api.App) {
	repo := repository.NewMaintenanceRepository(app.DB())
	svc := service.NewMaintenanceService(repo)
	h := handler.NewMaintenanceHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Permission: authz.Operation})
	g.GET("/vehicles/:id/maintenance", h.List)
	g.POST("/vehicles/:id/maintenance", h.Create)
	g.PUT("/maintenance/:id", h.Update)
	g.DELETE("/maintenance/:id", h.Delete)
}
