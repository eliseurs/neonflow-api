package route

import (
	"neonflow-api/internal/domain/fleet/handler"
	"neonflow-api/internal/domain/fleet/repository"
	"neonflow-api/internal/domain/fleet/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/authz"
)

// Register expõe a frota para a central (RN-01.2: cadastrada pela central).
func Register(app *api.App) {
	repo := repository.NewFleetRepository(app.DB())
	svc := service.NewFleetService(repo)
	h := handler.NewFleetHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Permission: authz.Operation})
	g.GET("/vehicles", h.List)
	g.GET("/vehicles/stats", h.Comparison)
	g.GET("/vehicles/:id/stats", h.Stats)
	g.GET("/vehicles/:id", h.Get)
	g.POST("/vehicles", h.Create)
	g.PUT("/vehicles/:id", h.Update)
}
