package route

import (
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/contract/handler"
	"neonflow-api/internal/domain/contract/repository"
	"neonflow-api/internal/domain/contract/service"
	"neonflow-api/internal/infra/api"
)

// Register expõe os contratos e a revisão jurídica. Só admin e revisora entram; a central não.
// Quem pode cada ação (criar versão, aprovar, publicar) é conferido no serviço, pelo papel do JWT.
func Register(app *api.App) {
	repo := repository.NewContractRepository(app.DB())
	svc := service.NewContractService(repo)
	h := handler.NewContractHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin, authentication.RoleLegalReviewer}})
	g.GET("/contracts", h.Documents)
	g.GET("/contracts/:code/versions", h.Versions)
	g.POST("/contracts/:code/versions", h.CreateVersion)
	g.GET("/contract-versions/:id", h.Version)
	g.POST("/contract-versions/:id/comments", h.AddComment)
	g.POST("/contract-versions/:id/events", h.AddEvent)
}
