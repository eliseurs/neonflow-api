package route

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/config"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/pkg/security"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Configs{JWTSecret: "s", JWTExpiryHours: 1}
	app := api.New(cfg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	Register(app)
	got := map[string]bool{}
	for _, r := range app.Router().Routes() {
		got[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{"GET /api/contracts", "GET /api/contracts/:code/versions", "POST /api/contracts/:code/versions",
		"GET /api/contract-versions/:id", "POST /api/contract-versions/:id/comments", "POST /api/contract-versions/:id/events"} {
		assert.True(t, got[want], want)
	}

	// Sem login: 401. Central logada: 403 (não participa da revisão).
	w := httptest.NewRecorder()
	app.Router().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/contracts", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	token := centralToken(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/contracts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	app.Router().ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func centralToken(t *testing.T, cfg config.Configs) string {
	t.Helper()
	tp := security.NewTokenProvider(cfg.JWTSecret, time.Hour)
	token, _, err := tp.Generate("u1", "central")
	require.NoError(t, err)
	return token
}
