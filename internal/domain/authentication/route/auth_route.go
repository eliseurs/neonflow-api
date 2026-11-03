package route

import (
	"neonflow-api/internal/domain/authentication/handler"
	"neonflow-api/internal/domain/authentication/repository"
	"neonflow-api/internal/domain/authentication/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/security"
)

func Register(app *api.App) {
	repo := repository.NewAuthRepository(app.DB())
	svc := service.NewAuthService(repo, security.NewBcryptHasher(), app.Tokens(), app.Logger())
	h := handler.NewAuthHandler(svc, app.Logger())

	app.API(api.Routes{}).POST("/auth/login", h.Login)
	app.API(api.Routes{Authorization: true}).GET("/auth/me", h.Me)
}
