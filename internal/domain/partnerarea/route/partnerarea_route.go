package route

import (
	"neonflow-api/internal/domain/partnerarea/handler"
	"neonflow-api/internal/domain/partnerarea/repository"
	"neonflow-api/internal/domain/partnerarea/service"
	transferRepo "neonflow-api/internal/domain/transfer/repository"
	transferService "neonflow-api/internal/domain/transfer/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/authz"
)

// Register expõe a área do parceiro logado (RN-07.8 a 07.11) e a oferta aberta do transfer, em que o
// motorista aceita ou desiste da corrida (PEN-74). O aceite usa o mesmo serviço da central.
func Register(app *api.App) {
	transfers := transferService.NewTransferService(transferRepo.NewTransferRepository(app.DB()))
	svc := service.NewPartnerAreaService(repository.NewPartnerAreaRepository(app.DB()), transfers)
	h := handler.NewPartnerAreaHandler(svc, app.Logger())
	g := app.API(api.Routes{Authorization: true, Permission: authz.PartnerArea})
	g.GET("/partner/area", h.Area)
	g.POST("/partner/rides/:id/accept", h.Accept)
	g.POST("/partner/rides/:id/release", h.Release)
}
