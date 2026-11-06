package route

import (
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/transfer/handler"
	"neonflow-api/internal/domain/transfer/repository"
	"neonflow-api/internal/domain/transfer/service"
	"neonflow-api/internal/infra/api"
)

// Register expõe o transfer para a central; motoristas ainda não entram no painel (RN-05.6).
func Register(app *api.App) {
	repo := repository.NewTransferRepository(app.DB())
	svc := service.NewTransferService(repo)
	h := handler.NewTransferHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin, authentication.RoleCentral}})
	g.GET("/transfers", h.List)
	g.POST("/transfers", h.Create)
	g.GET("/transfers/:id", h.Get)
	g.GET("/transfers/:id/events", h.Events)
	g.GET("/transfers/:id/drivers", h.Drivers)
	g.POST("/transfers/:id/offer", h.Offer)
	g.POST("/transfers/:id/notify", h.Notify)
	g.POST("/transfers/:id/accept", h.Accept)
	g.POST("/transfers/:id/release", h.Release)
	g.PATCH("/transfers/:id/paid", h.SetPaid)
	g.POST("/transfers/:id/start", h.Start)
	g.POST("/transfers/:id/complete", h.Complete)
	g.POST("/transfers/:id/cancel", h.Cancel)
}
