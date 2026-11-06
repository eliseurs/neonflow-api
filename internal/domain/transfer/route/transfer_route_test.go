package route

import (
	"io"
	"log/slog"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neonflow-api/internal/config"
	"neonflow-api/internal/infra/api"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := api.New(config.Configs{JWTSecret: "s", JWTExpiryHours: 1}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	Register(app)
	got := map[string]bool{}
	for _, r := range app.Router().Routes() {
		got[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{"GET /api/transfers", "POST /api/transfers", "GET /api/transfers/:id", "GET /api/transfers/:id/events", "GET /api/transfers/:id/drivers", "POST /api/transfers/:id/offer", "POST /api/transfers/:id/notify", "POST /api/transfers/:id/accept", "POST /api/transfers/:id/release", "PATCH /api/transfers/:id/paid", "POST /api/transfers/:id/start", "POST /api/transfers/:id/complete", "POST /api/transfers/:id/cancel"} {
		assert.True(t, got[want], want)
	}
}
