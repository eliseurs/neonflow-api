package route

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/config"
	"neonflow-api/internal/infra/api"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := api.New(config.Configs{JWTSecret: "s", JWTExpiryHours: 1}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	Register(app)

	w := httptest.NewRecorder()
	app.Router().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/partner/area", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Cliente e central não abrem a área do parceiro.
	for _, role := range []string{"cliente", "central"} {
		token, _, err := app.Tokens().Generate("u1", role)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/partner/area", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		app.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code, role)
	}
}
