package api

import (
	"context"
	"errors"
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
	"neonflow-api/internal/pkg/authz"
	"neonflow-api/internal/pkg/middleware"
)

type fakePinger struct{ err error }

func (f fakePinger) PingContext(context.Context) error { return f.err }

func newTestApp(t *testing.T) *App {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Configs{AppEnv: config.EnvLocal, JWTSecret: "s", JWTExpiryHours: 1, AllowedOrigins: []string{"http://localhost:5173"}}
	return New(cfg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func get(app *App, path string, header http.Header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if header != nil {
		req.Header = header
	}
	w := httptest.NewRecorder()
	app.Router().ServeHTTP(w, req)
	return w
}

func TestHealthCheck(t *testing.T) {
	app := newTestApp(t)
	app.HealthCheck()

	w := get(app, "/api/health", nil)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code, "sem banco configurado")
	assert.Contains(t, w.Body.String(), `"database":"unavailable"`)

	app.pinger = fakePinger{}
	for _, p := range []string{"/health", "/api/health"} {
		w = get(app, p, nil)
		assert.Equal(t, http.StatusOK, w.Code, p)
		assert.Contains(t, w.Body.String(), `"database":"ok"`)
	}

	app.pinger = fakePinger{err: errors.New("down")}
	assert.Equal(t, http.StatusServiceUnavailable, get(app, "/health", nil).Code)
}

func TestAPI_RouteGroups(t *testing.T) {
	app := newTestApp(t)
	ok := func(c *gin.Context) { Respond(c, http.StatusOK, gin.H{"user": c.GetString("userID")}) }
	app.API(Routes{}).GET("/public", ok)
	app.API(Routes{Authorization: true}).GET("/private", ok)
	app.API(Routes{Authorization: true, Permission: authz.Users}).GET("/admin", ok)

	assert.Equal(t, http.StatusOK, get(app, "/api/public", nil).Code)
	assert.Equal(t, http.StatusUnauthorized, get(app, "/api/private", nil).Code)

	central, _, err := app.Tokens().Generate("u1", "central")
	require.NoError(t, err)
	h := http.Header{"Authorization": {"Bearer " + central}}
	w := get(app, "/api/private", h)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":200,"data":{"user":"u1"}}`, w.Body.String())
	assert.Equal(t, http.StatusForbidden, get(app, "/api/admin", h).Code)

	assert.Equal(t, http.StatusNotFound, get(app, "/nao-existe", nil).Code)
}

type fakeStore struct {
	role    string
	entries []middleware.AuditEntry
}

func (f *fakeStore) ActiveRole(context.Context, string) (string, error) {
	if f.role == "" {
		return "", middleware.ErrAccountNotFound
	}
	return f.role, nil
}

func (f *fakeStore) Record(_ context.Context, e middleware.AuditEntry) error {
	f.entries = append(f.entries, e)
	return nil
}

// Com banco, o papel vem do usuário atual e as rotas são auditadas (RN-05.8, RN-05.9).
func TestAPI_AccountsAndAudit(t *testing.T) {
	app := newTestApp(t)
	st := &fakeStore{role: "central"}
	app.accounts, app.audit = st, st
	ok := func(c *gin.Context) { Respond(c, http.StatusOK, gin.H{"role": c.GetString("role")}) }
	app.API(Routes{Authorization: true, Permission: authz.Users}).GET("/admin", ok)

	// O token ainda diz admin, mas o usuário foi rebaixado para central.
	token, _, err := app.Tokens().Generate("u1", "admin")
	require.NoError(t, err)
	h := http.Header{"Authorization": {"Bearer " + token}}
	assert.Equal(t, http.StatusForbidden, get(app, "/api/admin", h).Code)
	assert.Equal(t, []middleware.AuditEntry{{UserID: "u1", Role: "central", Method: "GET", Route: "/api/admin", Path: "/api/admin", Status: http.StatusForbidden}}, st.entries)

	st.role = "admin"
	assert.Equal(t, http.StatusOK, get(app, "/api/admin", h).Code)

	st.role = ""
	assert.Equal(t, http.StatusUnauthorized, get(app, "/api/admin", h).Code, "usuário desativado")
	assert.Len(t, st.entries, 1)
}

func TestRespondError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	RespondError(c, http.StatusBadRequest, "ruim")
	assert.JSONEq(t, `{"status":400,"data":{"error":"ruim"}}`, w.Body.String())
}

func TestAccessorsAndProductionMode(t *testing.T) {
	app := newTestApp(t)
	assert.Nil(t, app.DB())
	assert.NotNil(t, app.Logger())
	assert.Equal(t, config.EnvLocal, app.Configs().AppEnv)

	prod := New(config.Configs{AppEnv: config.EnvHml, JWTSecret: "s", JWTExpiryHours: 1}, nil, app.Logger())
	assert.Equal(t, gin.ReleaseMode, gin.Mode())
	assert.NotNil(t, prod.Router())
	gin.SetMode(gin.TestMode)
}

func TestServe_GracefulShutdown(t *testing.T) {
	app := newTestApp(t)
	app.configs.Port = "0"
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.serve(ctx) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server não encerrou")
	}
}

func TestServe_ListenError(t *testing.T) {
	app := newTestApp(t)
	app.configs.Port = "porta-invalida"
	err := app.serve(context.Background())
	assert.Error(t, err)
}
