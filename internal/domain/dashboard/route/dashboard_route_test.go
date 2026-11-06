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
	for _, want := range []string{"GET /api/dashboard"} {
		assert.True(t, got[want], want)
	}
}
