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
	"neonflow-api/internal/domain/user"
)

type fakeService struct {
	err   error
	actor *string
}

func (f fakeService) List(context.Context) ([]user.User, error) {
	return []user.User{{ID: "u1"}}, f.err
}
func (f fakeService) Create(_ context.Context, r user.CreateRequest) (user.User, error) {
	return user.User{ID: "new", Email: r.Email}, f.err
}
func (f fakeService) Update(_ context.Context, actor, id string, r user.UpdateRequest) (user.User, error) {
	*f.actor = actor
	return user.User{ID: id, Name: r.Name}, f.err
}
func (f fakeService) ResetPassword(context.Context, string, user.PasswordRequest) error { return f.err }
func (f fakeService) ChangeOwnPassword(_ context.Context, actor string, _ user.ChangeOwnPasswordRequest) error {
	*f.actor = actor
	return f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc user.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewUserHandler(svc, nopLog{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "admin1") })
	r.GET("/users", h.List)
	r.POST("/users", h.Create)
	r.PUT("/users/:id", h.Update)
	r.POST("/users/:id/password", h.ResetPassword)
	r.POST("/auth/password", h.ChangeOwnPassword)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUserHandler_OK(t *testing.T) {
	var actor string
	svc := fakeService{actor: &actor}
	assert.Equal(t, http.StatusOK, do(svc, http.MethodGet, "/users", "").Code)
	w := do(svc, http.MethodPost, "/users", `{"email":"ana@neonflow.test"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "ana@neonflow.test")
	w = do(svc, http.MethodPut, "/users/u2", `{"name":"Ana","role":"central","active":false}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "admin1", actor, "quem edita vem do token")
	assert.Equal(t, http.StatusOK, do(svc, http.MethodPost, "/users/u2/password", `{"password":"x"}`).Code)
	actor = ""
	assert.Equal(t, http.StatusOK, do(svc, http.MethodPost, "/auth/password", `{"current_password":"a","new_password":"b"}`).Code)
	assert.Equal(t, "admin1", actor)
}

func TestUserHandler_BadBody(t *testing.T) {
	var actor string
	svc := fakeService{actor: &actor}
	for _, c := range [][2]string{{http.MethodPost, "/users"}, {http.MethodPut, "/users/u2"}, {http.MethodPost, "/users/u2/password"}, {http.MethodPost, "/auth/password"}} {
		assert.Equal(t, http.StatusBadRequest, do(svc, c[0], c[1], `{`).Code, c[1])
	}
}

func TestUserHandler_Errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{authentication.ErrUserNotFound, http.StatusNotFound},
		{authentication.ErrEmailTaken, http.StatusConflict},
		{user.ErrSelfLockout, http.StatusConflict},
		{authentication.ErrNameRequired, http.StatusBadRequest},
		{authentication.ErrInvalidEmail, http.StatusBadRequest},
		{authentication.ErrWeakPassword, http.StatusBadRequest},
		{authentication.ErrInvalidRole, http.StatusBadRequest},
		{user.ErrWrongPassword, http.StatusBadRequest},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		var actor string
		svc := fakeService{err: tc.err, actor: &actor}
		for _, c := range [][3]string{
			{http.MethodGet, "/users", ""},
			{http.MethodPost, "/users", `{}`},
			{http.MethodPut, "/users/u2", `{}`},
			{http.MethodPost, "/users/u2/password", `{}`},
			{http.MethodPost, "/auth/password", `{}`},
		} {
			w := do(svc, c[0], c[1], c[2])
			assert.Equal(t, tc.status, w.Code, tc.err.Error()+" "+c[1])
		}
	}
}
