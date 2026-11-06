package route

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	for _, want := range []string{"GET /api/terms", "POST /api/terms", "POST /api/bookings/:id/terms-link", "GET /api/bookings/:id/terms",
		"GET /api/public/terms/:token", "POST /api/public/terms/:token/accept"} {
		assert.True(t, got[want], want)
	}
	// Publicar exige login de admin.
	w := httptest.NewRecorder()
	app.Router().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/terms", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
