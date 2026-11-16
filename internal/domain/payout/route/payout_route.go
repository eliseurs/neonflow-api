package route

import (
	"neonflow-api/internal/domain/payout/handler"
	"neonflow-api/internal/domain/payout/repository"
	"neonflow-api/internal/domain/payout/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/authz"
)

// Register expõe os pagamentos a parceiros (comissão e repasse) para a central (RN-03.12, RN-03.13).
func Register(app *api.App) {
	repo := repository.NewPayoutRepository(app.DB())
	svc := service.NewPayoutService(repo)
	h := handler.NewPayoutHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Permission: authz.Operation})
	g.GET("/payouts", h.List)
	g.POST("/payouts/:id/pay", h.Pay)
}
