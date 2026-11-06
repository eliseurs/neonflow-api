package route

import (
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/partner/handler"
	"neonflow-api/internal/domain/partner/repository"
	"neonflow-api/internal/domain/partner/service"
	"neonflow-api/internal/infra/api"
)

// Register expõe o cadastro de parceiros para a central (RN-07; parceiro ainda não entra no painel, RN-05.6).
func Register(app *api.App) {
	repo := repository.NewPartnerRepository(app.DB())
	svc := service.NewPartnerService(repo)
	h := handler.NewPartnerHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin, authentication.RoleCentral}})
	g.GET("/partners", h.List)
	g.GET("/partners/code/:code", h.ByCode)
	g.GET("/partners/:id", h.Get)
	g.POST("/partners", h.Create)
	g.PUT("/partners/:id", h.Update)
}
