package route

import (
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/customer/handler"
	"neonflow-api/internal/domain/customer/repository"
	"neonflow-api/internal/domain/customer/service"
	"neonflow-api/internal/infra/api"
)

// Register expõe o cadastro de clientes para a central; anonimizar é só do admin (encarregado, RN-04.5).
func Register(app *api.App) {
	repo := repository.NewCustomerRepository(app.DB())
	svc := service.NewCustomerService(repo)
	h := handler.NewCustomerHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin, authentication.RoleCentral}})
	g.GET("/customers", h.List)
	g.GET("/customers/:id", h.Get)
	g.POST("/customers", h.Create)
	g.PUT("/customers/:id", h.Update)

	admin := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin}})
	admin.POST("/customers/:id/anonymize", h.Anonymize)
}
