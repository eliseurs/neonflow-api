package route

import (
	"neonflow-api/internal/domain/partnerarea/handler"
	"neonflow-api/internal/domain/partnerarea/repository"
	"neonflow-api/internal/domain/partnerarea/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/authz"
)

// Register expõe a área do parceiro logado, só leitura (RN-07.8 a 07.11).
func Register(app *api.App) {
	svc := service.NewPartnerAreaService(repository.NewPartnerAreaRepository(app.DB()))
	h := handler.NewPartnerAreaHandler(svc, app.Logger())
	app.API(api.Routes{Authorization: true, Permission: authz.PartnerArea}).GET("/partner/area", h.Area)
}
