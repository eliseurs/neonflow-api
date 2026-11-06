package route

import (
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/user/handler"
	"neonflow-api/internal/domain/user/repository"
	"neonflow-api/internal/domain/user/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/security"
)

// Register expõe a gestão de usuários só para o admin, e a troca da própria senha para todos (RN-05.6).
func Register(app *api.App) {
	repo := repository.NewUserRepository(app.DB())
	svc := service.NewUserService(repo, security.NewBcryptHasher())
	h := handler.NewUserHandler(svc, app.Logger())

	admin := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin}})
	admin.GET("/users", h.List)
	admin.POST("/users", h.Create)
	admin.PUT("/users/:id", h.Update)
	admin.POST("/users/:id/password", h.ResetPassword)

	app.API(api.Routes{Authorization: true}).POST("/auth/password", h.ChangeOwnPassword)
}
