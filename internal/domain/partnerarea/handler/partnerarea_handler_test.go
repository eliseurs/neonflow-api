package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neonflow-api/internal/domain/partnerarea"
)

type fakeService struct {
	err  error
	user string
}

func (f *fakeService) Area(_ context.Context, user string) (partnerarea.Area, error) {
	f.user = user
	return partnerarea.Area{Partner: partnerarea.Partner{Name: "Pedro"}}, f.err
}

type spyLog struct{ n int }

func (s *spyLog) Error(string, ...any) { s.n++ }

func do(svc partnerarea.Service, log partnerarea.Logger) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.GET("/partner/area", NewPartnerAreaHandler(svc, log).Area)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/partner/area", nil))
	return w
}

func TestArea(t *testing.T) {
	svc := &fakeService{}
	w := do(svc, &spyLog{})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"Pedro"`)
	assert.Equal(t, "u1", svc.user, "o parceiro vem da conta logada")

	assert.Equal(t, http.StatusForbidden, do(&fakeService{err: partnerarea.ErrNoPartner}, &spyLog{}).Code)
	log := &spyLog{}
	assert.Equal(t, http.StatusInternalServerError, do(&fakeService{err: errors.New("db")}, log).Code)
	assert.Equal(t, 1, log.n)
}
