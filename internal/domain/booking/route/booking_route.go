package route

import (
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/booking/handler"
	"neonflow-api/internal/domain/booking/repository"
	"neonflow-api/internal/domain/booking/service"
	"neonflow-api/internal/infra/api"
)

// Register expõe as reservas para a central. Autosserviço do cliente está fora do escopo.
func Register(app *api.App) {
	repo := repository.NewBookingRepository(app.DB())
	svc := service.NewBookingService(repo)
	h := handler.NewBookingHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin, authentication.RoleCentral}})
	g.GET("/availability", h.Available)
	g.GET("/bookings", h.List)
	g.GET("/bookings/:id", h.Get)
	g.GET("/bookings/:id/events", h.Events)
	g.POST("/bookings", h.Create)
	g.PATCH("/bookings/:id/checks", h.SetChecks)
	g.POST("/bookings/:id/advance", h.Advance)
	g.POST("/bookings/:id/cancel", h.Cancel)
}
