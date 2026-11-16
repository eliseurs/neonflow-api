package route

import (
	"neonflow-api/internal/domain/audit/handler"
	"neonflow-api/internal/domain/audit/repository"
	"neonflow-api/internal/domain/audit/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/authz"
)

// Register expõe a trilha de auditoria a quem tem a permissão auditoria (admin, RN-05.9).
func Register(app *api.App) {
	repo := repository.NewAuditRepository(app.DB())
	h := handler.NewAuditHandler(service.NewAuditService(repo), app.Logger())

	app.API(api.Routes{Authorization: true, Permission: authz.Audit}).GET("/audit", h.List)
}
