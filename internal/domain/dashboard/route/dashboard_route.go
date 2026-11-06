package route

import (
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/dashboard/handler"
	"neonflow-api/internal/domain/dashboard/repository"
	"neonflow-api/internal/domain/dashboard/service"
	"neonflow-api/internal/infra/api"
)

// Register expõe os indicadores da tela inicial para a central (PEN-11).
func Register(app *api.App) {
	repo := repository.NewDashboardRepository(app.DB())
	svc := service.NewDashboardService(repo)
	h := handler.NewDashboardHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin, authentication.RoleCentral}})
	g.GET("/dashboard", h.Summary)
}
