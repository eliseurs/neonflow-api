package route

import (
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/checklist/handler"
	"neonflow-api/internal/domain/checklist/repository"
	"neonflow-api/internal/domain/checklist/service"
	"neonflow-api/internal/infra/api"
)

// Register expõe o checklist de devolução para a central configurar (RN-01.6).
func Register(app *api.App) {
	repo := repository.NewChecklistRepository(app.DB())
	svc := service.NewChecklistService(repo)
	h := handler.NewChecklistHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin, authentication.RoleCentral}})
	g.GET("/checklist-items", h.List)
	g.POST("/checklist-items", h.Create)
	g.PUT("/checklist-items/:id", h.Update)
}
