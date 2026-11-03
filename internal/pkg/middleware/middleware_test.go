package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neonflow-api/internal/pkg/security"
)

type fakeParser struct{}

func (fakeParser) Parse(token string) (security.Claims, error) {
	if token == "ok" {
		return security.Claims{UserID: "u1", Role: "central"}, nil
	}
	return security.Claims{}, errors.New("bad")
}

func init() { gin.SetMode(gin.TestMode) }

func serve(r *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthMiddleware(t *testing.T) {
	r := gin.New()
	r.GET("/x", AuthMiddleware(fakeParser{}), func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("userID")+":"+c.GetString("role"))
	})

	cases := []struct {
		name   string
		header string
		status int
	}{
		{"sem header", "", http.StatusUnauthorized},
		{"esquema errado", "Basic ok", http.StatusUnauthorized},
		{"bearer vazio", "Bearer ", http.StatusUnauthorized},
		{"token inválido", "Bearer ruim", http.StatusUnauthorized},
		{"token válido", "Bearer ok", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := serve(r, http.MethodGet, "/x", map[string]string{"Authorization": tc.header})
			assert.Equal(t, tc.status, w.Code)
			if tc.status == http.StatusOK {
				assert.Equal(t, "u1:central", w.Body.String())
			} else {
				assert.Contains(t, w.Body.String(), `"error"`)
			}
		})
	}
}

func TestRoleGuard(t *testing.T) {
	r := gin.New()
	setRole := func(role string) gin.HandlerFunc { return func(c *gin.Context) { c.Set("role", role) } }
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/admin", setRole("admin"), RoleGuard("admin"), ok)
	r.GET("/central", setRole("central"), RoleGuard("admin"), ok)

	assert.Equal(t, http.StatusOK, serve(r, http.MethodGet, "/admin", nil).Code)
	assert.Equal(t, http.StatusForbidden, serve(r, http.MethodGet, "/central", nil).Code)
}

func TestCORS(t *testing.T) {
	r := gin.New()
	r.Use(CORS([]string{"http://localhost:5173"}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := serve(r, http.MethodGet, "/x", map[string]string{"Origin": "http://localhost:5173"})
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))

	w = serve(r, http.MethodGet, "/x", map[string]string{"Origin": "https://evil.example"})
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))

	w = serve(r, http.MethodOptions, "/x", map[string]string{"Origin": "http://localhost:5173"})
	assert.Equal(t, http.StatusNoContent, w.Code)
}
