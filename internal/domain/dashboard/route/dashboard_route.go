package route

import (
	"neonflow-api/internal/domain/dashboard/handler"
	"neonflow-api/internal/domain/dashboard/repository"
	"neonflow-api/internal/domain/dashboard/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/authz"
)

// Register expõe os indicadores da tela inicial para a central (PEN-11).
func Register(app *api.App) {
	repo := repository.NewDashboardRepository(app.DB())
	svc := service.NewDashboardService(repo)
	h := handler.NewDashboardHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Permission: authz.Operation})
	g.GET("/dashboard", h.Summary)
}
