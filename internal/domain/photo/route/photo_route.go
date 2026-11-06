package route

import (
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/photo"
	"neonflow-api/internal/domain/photo/handler"
	"neonflow-api/internal/domain/photo/repository"
	"neonflow-api/internal/domain/photo/service"
	"neonflow-api/internal/infra/api"
)

// Register expõe as fotos do veículo e da reserva para a central (RN-01.6, RN-01.7, ADR-0014).
func Register(app *api.App, store photo.Store) {
	repo := repository.NewPhotoRepository(app.DB())
	svc := service.NewPhotoService(repo, store)
	h := handler.NewPhotoHandler(svc, app.Logger())

	g := app.API(api.Routes{Authorization: true, Roles: []string{authentication.RoleAdmin, authentication.RoleCentral}})
	g.GET("/vehicles/:id/photos", h.ListVehicle)
	g.POST("/vehicles/:id/photos", h.UploadVehicle)
	g.GET("/bookings/:id/photos", h.ListBooking)
	g.POST("/bookings/:id/photos", h.UploadBooking)
	g.DELETE("/photos/:id", h.Delete)
	g.POST("/photos/:id/cover", h.Cover)

	// Site público: só a frota e as fotos de veículo, nunca as da reserva.
	public := app.API(api.Routes{})
	public.GET("/public/fleet", h.PublicFleet)
	public.GET("/public/photos/:id/w/:width", h.Variant)
}
