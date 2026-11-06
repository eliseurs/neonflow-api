package route

import (
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/terms/handler"
	"neonflow-api/internal/domain/terms/repository"
	"neonflow-api/internal/domain/terms/service"
	"neonflow-api/internal/infra/api"
)

// Register expõe os termos: o admin publica, a central gera o link, o cliente aceita sem login (RN-02.3).
func Register(app *api.App) {
	repo := repository.NewTermsRepository(app.DB())
	svc := service.NewTermsService(repo)
	h := handler.NewTermsHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin, authentication.RoleCentral}})
	g.GET("/terms", h.Versions)
	g.POST("/bookings/:id/terms-link", h.CreateLink)
	g.GET("/bookings/:id/terms", h.BookingAcceptance)

	admin := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin}})
	admin.POST("/terms", h.Publish)

	// Sem login: o código do link é a autorização (32 bytes aleatórios, só o hash no banco, 7 dias).
	public := app.API(api.Routes{})
	public.GET("/public/terms/:token", h.Public)
	public.POST("/public/terms/:token/accept", h.Accept)
}
