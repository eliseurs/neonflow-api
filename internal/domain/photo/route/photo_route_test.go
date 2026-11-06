package route

import (
	"io"
	"log/slog"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neonflow-api/internal/config"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/infra/storage"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := api.New(config.Configs{JWTSecret: "s", JWTExpiryHours: 1}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	Register(app, storage.Disabled{})
	got := map[string]bool{}
	for _, r := range app.Router().Routes() {
		got[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{"GET /api/vehicles/:id/photos", "POST /api/vehicles/:id/photos", "GET /api/bookings/:id/photos",
		"POST /api/bookings/:id/photos", "DELETE /api/photos/:id", "POST /api/photos/:id/cover",
		"GET /api/public/fleet", "GET /api/public/photos/:id/w/:width"} {
		assert.True(t, got[want], want)
	}
}
