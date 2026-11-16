package route

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	for _, want := range []string{"POST /api/partners/:id/invite", "POST /api/public/accounts/signup", "POST /api/public/accounts/confirm",
		"POST /api/public/accounts/forgot-password", "POST /api/public/accounts/reset-password", "POST /api/public/accounts/accept-invite"} {
		assert.True(t, got[want], want)
	}

	// Convidar exige login da central.
	w := httptest.NewRecorder()
	app.Router().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/partners/p1/invite", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// O cadastro é público; sem o envio configurado, responde 503 antes de tocar no banco.
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/public/accounts/signup", strings.NewReader(`{"name":"Ana","email":"a@neonflow.test","password":"senha-forte-1"}`))
	req.Header.Set("Content-Type", "application/json")
	app.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "envio de e-mail não configurado")
}
