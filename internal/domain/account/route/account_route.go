package route

import (
	"neonflow-api/internal/domain/account"
	"neonflow-api/internal/domain/account/handler"
	"neonflow-api/internal/domain/account/repository"
	"neonflow-api/internal/domain/account/service"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/authz"
	"neonflow-api/internal/pkg/security"
)

// Register expõe as contas de parceiros e clientes (RN-05.11 a RN-05.14): a central convida o parceiro;
// cadastro, confirmação, nova senha e aceite do convite são públicos, e o código do link é a autorização.
func Register(app *api.App, mail account.Mailer) {
	repo := repository.NewAccountRepository(app.DB())
	svc := service.NewAccountService(repo, security.NewBcryptHasher(), mail, app.Logger(), app.Configs().AppURL)
	h := handler.NewAccountHandler(svc, app.Logger())

	app.API(api.Routes{Authorization: true, Permission: authz.Operation}).POST("/partners/:id/invite", h.InvitePartner)

	public := app.API(api.Routes{})
	public.POST("/public/accounts/signup", h.Signup)
	public.POST("/public/accounts/confirm", h.Confirm)
	public.POST("/public/accounts/forgot-password", h.ForgotPassword)
	public.POST("/public/accounts/reset-password", h.ResetPassword)
	public.POST("/public/accounts/accept-invite", h.AcceptInvite)
}
