package route

import (
	bookingRepo "neonflow-api/internal/domain/booking/repository"
	bookingService "neonflow-api/internal/domain/booking/service"
	"neonflow-api/internal/domain/siterequest"
	"neonflow-api/internal/domain/siterequest/handler"
	"neonflow-api/internal/domain/siterequest/repository"
	"neonflow-api/internal/domain/siterequest/service"
	transferRepo "neonflow-api/internal/domain/transfer/repository"
	transferService "neonflow-api/internal/domain/transfer/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/authz"
)

// Register expõe os pedidos do site (RN-02.33 a 02.37): o cliente logado pede e acompanha; a central
// confirma, recusa ou pede ajuste. A confirmação usa os mesmos serviços de reserva e de transfer da central.
func Register(app *api.App, mail siterequest.Mailer) {
	bookings := bookingService.NewBookingService(bookingRepo.NewBookingRepository(app.DB()))
	transfers := transferService.NewTransferService(transferRepo.NewTransferRepository(app.DB()))
	svc := service.NewSiteRequestService(repository.NewSiteRequestRepository(app.DB()), bookings, transfers, mail, app.Logger(), app.Configs().AppURL)
	h := handler.NewSiteRequestHandler(svc, app.Logger())

	client := app.API(api.Routes{Authorization: true, Permission: authz.CustomerArea})
	client.GET("/client/cars", h.Cars)
	client.GET("/client/requests", h.Mine)
	client.POST("/client/requests", h.Create)
	client.GET("/client/requests/:id", h.MineDetail)
	client.PUT("/client/requests/:id", h.Resubmit)
	client.POST("/client/requests/:id/cancel", h.CancelMine)

	central := app.API(api.Routes{Authorization: true, Permission: authz.Operation})
	central.GET("/requests", h.List)
	central.GET("/requests/:id", h.Get)
	central.POST("/requests/:id/confirm", h.Confirm)
	central.POST("/requests/:id/adjust", h.Adjust)
	central.POST("/requests/:id/refuse", h.Refuse)
}
