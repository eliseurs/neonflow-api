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

	"neonflow-api/internal/domain/contract"
)

type fakeService struct {
	err   error
	actor contract.Actor
}

func (f *fakeService) Documents(context.Context) ([]contract.Document, error) { return nil, f.err }
func (f *fakeService) Versions(context.Context, string) ([]contract.Summary, error) {
	return nil, f.err
}
func (f *fakeService) CreateVersion(_ context.Context, _ string, _ contract.VersionRequest, a contract.Actor) (contract.Summary, error) {
	f.actor = a
	return contract.Summary{}, f.err
}
func (f *fakeService) Version(context.Context, string) (contract.Version, error) {
	return contract.Version{}, f.err
}
func (f *fakeService) AddComment(_ context.Context, _ string, _ contract.CommentRequest, a contract.Actor) (contract.Comment, error) {
	f.actor = a
	return contract.Comment{}, f.err
}
func (f *fakeService) AddEvent(_ context.Context, _ string, _ contract.EventRequest, a contract.Actor) (contract.Event, error) {
	f.actor = a
	return contract.Event{}, f.err
}

type spyLog struct{ msgs []string }

func (s *spyLog) Error(msg string, _ ...any) { s.msgs = append(s.msgs, msg) }

func do(svc contract.Service, log contract.Logger, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewContractHandler(svc, log)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1"); c.Set("role", "legal_reviewer") })
	r.GET("/contracts", h.Documents)
	r.GET("/contracts/:code/versions", h.Versions)
	r.POST("/contracts/:code/versions", h.CreateVersion)
	r.GET("/contract-versions/:id", h.Version)
	r.POST("/contract-versions/:id/comments", h.AddComment)
	r.POST("/contract-versions/:id/events", h.AddEvent)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

func TestHandlerSuccess(t *testing.T) {
	cases := []struct {
		method, path, body string
		status             int
	}{
		{http.MethodGet, "/contracts", "", http.StatusOK},
		{http.MethodGet, "/contracts/01/versions", "", http.StatusOK},
		{http.MethodPost, "/contracts/01/versions", `{"body":"x"}`, http.StatusCreated},
		{http.MethodGet, "/contract-versions/v", "", http.StatusOK},
		{http.MethodPost, "/contract-versions/v/comments", `{"clause":"1"}`, http.StatusCreated},
		{http.MethodPost, "/contract-versions/v/events", `{"action":"aprovar"}`, http.StatusCreated},
	}
	for _, tc := range cases {
		svc := &fakeService{}
		w := do(svc, &spyLog{}, tc.method, tc.path, tc.body)
		assert.Equal(t, tc.status, w.Code, tc.path)
	}
	svc := &fakeService{}
	do(svc, &spyLog{}, http.MethodPost, "/contract-versions/v/events", `{"action":"aprovar","user_id":"outro"}`)
	assert.Equal(t, contract.Actor{UserID: "u1", Role: "legal_reviewer"}, svc.actor, "autor vem do JWT, nunca do corpo")
}

func TestHandlerBadJSON(t *testing.T) {
	for _, path := range []string{"/contracts/01/versions", "/contract-versions/v/comments", "/contract-versions/v/events"} {
		w := do(&fakeService{}, &spyLog{}, http.MethodPost, path, "{")
		assert.Equal(t, http.StatusBadRequest, w.Code, path)
	}
}

func TestHandlerErrors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{contract.ErrDocumentNotFound, http.StatusNotFound},
		{contract.ErrVersionNotFound, http.StatusNotFound},
		{contract.ErrForbiddenAction, http.StatusForbidden},
		{contract.ErrForbiddenComment, http.StatusForbidden},
		{contract.ErrWrongStatus, http.StatusConflict},
		{contract.ErrHashMismatch, http.StatusConflict},
		{contract.ErrConfirmRequired, http.StatusBadRequest},
		{contract.ErrInvalidAction, http.StatusBadRequest},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		log := &spyLog{}
		w := do(&fakeService{err: tc.err}, log, http.MethodPost, "/contract-versions/v/events", `{}`)
		assert.Equal(t, tc.status, w.Code, tc.err.Error())
		if tc.status == http.StatusInternalServerError {
			assert.Len(t, log.msgs, 1)
		}
	}
}
