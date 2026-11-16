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
	"neonflow-api/internal/infra/email"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := api.New(config.Configs{JWTSecret: "s", JWTExpiryHours: 1}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	Register(app, email.Disabled{})
	got := map[string]bool{}
	for _, r := range app.Router().Routes() {
		got[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{"GET /api/client/cars", "GET /api/client/requests", "POST /api/client/requests", "GET /api/client/requests/:id",
		"PUT /api/client/requests/:id", "POST /api/client/requests/:id/cancel", "GET /api/requests", "GET /api/requests/:id",
		"POST /api/requests/:id/confirm", "POST /api/requests/:id/adjust", "POST /api/requests/:id/refuse"} {
		assert.True(t, got[want], want)
	}

	// Cliente e central exigem login.
	for _, path := range []string{"/api/client/requests", "/api/requests"} {
		w := httptest.NewRecorder()
		app.Router().ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code, path)
	}

	// Com token de cliente, a lista da central é negada; com token da central, a área do cliente também.
	for _, tc := range []struct{ role, path string }{{"cliente", "/api/requests"}, {"central", "/api/client/requests"}} {
		token, _, err := app.Tokens().Generate("u1", tc.role)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		app.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code, tc.role+" "+tc.path)
	}
}
