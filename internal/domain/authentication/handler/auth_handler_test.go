package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neonflow-api/internal/domain/authentication"
)

type fakeService struct {
	loginErr error
	meErr    error
}

func (f fakeService) Login(_ context.Context, req authentication.LoginRequest) (authentication.LoginResponse, error) {
	if f.loginErr != nil {
		return authentication.LoginResponse{}, f.loginErr
	}
	return authentication.LoginResponse{Token: "tok", User: authentication.User{ID: "u1", Email: req.Email}}, nil
}

func (f fakeService) Me(_ context.Context, id string) (authentication.User, error) {
	if f.meErr != nil {
		return authentication.User{}, f.meErr
	}
	return authentication.User{ID: id, Name: "Central"}, nil
}

func (fakeService) CreateUser(context.Context, authentication.CreateUserRequest) (authentication.User, error) {
	return authentication.User{}, nil
}

type nopLog struct{}

func (nopLog) Warn(string, ...any)  {}
func (nopLog) Error(string, ...any) {}

func router(svc authentication.Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(svc, nopLog{})
	r := gin.New()
	r.POST("/login", h.Login)
	r.GET("/me", func(c *gin.Context) {
		if userID != "" {
			c.Set("userID", userID)
		}
		h.Me(c)
	})
	return r
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLogin(t *testing.T) {
	cases := []struct {
		name   string
		svc    fakeService
		body   string
		status int
		want   string
	}{
		{"ok", fakeService{}, `{"email":"a@b.test","password":"x"}`, http.StatusOK, `"token":"tok"`},
		{"json inválido", fakeService{}, `{`, http.StatusBadRequest, "corpo da requisição inválido"},
		{"credencial inválida", fakeService{loginErr: authentication.ErrInvalidCredentials}, `{"email":"a@b.test","password":"x"}`, http.StatusUnauthorized, "e-mail ou senha inválidos"},
		{"erro interno", fakeService{loginErr: errors.New("db")}, `{"email":"a@b.test","password":"x"}`, http.StatusInternalServerError, "erro interno"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := do(router(tc.svc, ""), http.MethodPost, "/login", tc.body)
			assert.Equal(t, tc.status, w.Code)
			assert.Contains(t, w.Body.String(), tc.want)
		})
	}
}

func TestMe(t *testing.T) {
	w := do(router(fakeService{}, "u1"), http.MethodGet, "/me", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"u1"`)

	assert.Equal(t, http.StatusUnauthorized, do(router(fakeService{}, ""), http.MethodGet, "/me", "").Code)
	// Usuário desativado ou removido: 401, e o painel volta ao login.
	assert.Equal(t, http.StatusUnauthorized, do(router(fakeService{meErr: authentication.ErrUserNotFound}, "u1"), http.MethodGet, "/me", "").Code)
	assert.Equal(t, http.StatusInternalServerError, do(router(fakeService{meErr: errors.New("db")}, "u1"), http.MethodGet, "/me", "").Code)
}
