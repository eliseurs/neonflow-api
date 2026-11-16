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

	"neonflow-api/internal/domain/account"
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/infra/email"
)

type fakeService struct {
	err     error
	partner string
}

func (f *fakeService) Signup(context.Context, account.SignupRequest) error { return f.err }
func (f *fakeService) ConfirmEmail(context.Context, account.TokenRequest) (account.Confirmation, error) {
	return account.Confirmation{Confirmed: true, RequestsSent: 1, PasswordToken: "pw"}, f.err
}
func (f *fakeService) ForgotPassword(context.Context, account.EmailRequest) error   { return f.err }
func (f *fakeService) ResetPassword(context.Context, account.PasswordRequest) error { return f.err }
func (f *fakeService) AcceptInvite(context.Context, account.PasswordRequest) error  { return f.err }
func (f *fakeService) InvitePartner(_ context.Context, id string) (account.Invite, error) {
	f.partner = id
	return account.Invite{SentTo: "p@neonflow.test"}, f.err
}

type spyLog struct{ msgs []string }

func (s *spyLog) Error(msg string, _ ...any) { s.msgs = append(s.msgs, msg) }

func do(svc account.Service, log account.Logger, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewAccountHandler(svc, log)
	r := gin.New()
	r.POST("/signup", h.Signup)
	r.POST("/confirm", h.Confirm)
	r.POST("/forgot", h.ForgotPassword)
	r.POST("/reset", h.ResetPassword)
	r.POST("/accept", h.AcceptInvite)
	r.POST("/partners/:id/invite", h.InvitePartner)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_Success(t *testing.T) {
	cases := []struct {
		path, body string
		status     int
		want       string
	}{
		{"/signup", `{"name":"Ana","email":"a@neonflow.test","password":"senha-forte-1"}`, http.StatusAccepted, "enviamos um link"},
		{"/confirm", `{"token":"abc"}`, http.StatusOK, `"requests_sent":1,"password_token":"pw"`},
		{"/forgot", `{"email":"a@neonflow.test"}`, http.StatusAccepted, "enviamos um link"},
		{"/reset", `{"token":"abc","password":"senha-forte-1"}`, http.StatusOK, `"password_changed":true`},
		{"/accept", `{"token":"abc","password":"senha-forte-1"}`, http.StatusCreated, `"account_created":true`},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			w := do(&fakeService{}, &spyLog{}, http.MethodPost, tc.path, tc.body)
			assert.Equal(t, tc.status, w.Code)
			assert.Contains(t, w.Body.String(), tc.want)

			w = do(&fakeService{}, &spyLog{}, http.MethodPost, tc.path, `{`)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}

	svc := &fakeService{}
	w := do(svc, &spyLog{}, http.MethodPost, "/partners/p1/invite", "")
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"sent_to":"p@neonflow.test"`)
	assert.Equal(t, "p1", svc.partner)
}

func TestHandler_Errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{authentication.ErrNameRequired, http.StatusBadRequest},
		{authentication.ErrInvalidEmail, http.StatusBadRequest},
		{authentication.ErrWeakPassword, http.StatusBadRequest},
		{account.ErrInvalidToken, http.StatusBadRequest},
		{account.ErrPartnerNotFound, http.StatusNotFound},
		{account.ErrPartnerInactive, http.StatusConflict},
		{account.ErrPartnerHasAccount, http.StatusConflict},
		{account.ErrPartnerWithoutEmail, http.StatusConflict},
		{authentication.ErrEmailTaken, http.StatusConflict},
		{email.ErrNotConfigured, http.StatusServiceUnavailable},
		{account.ErrSendFailed, http.StatusBadGateway},
	}
	for _, tc := range cases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			w := do(&fakeService{err: tc.err}, &spyLog{}, http.MethodPost, "/partners/p1/invite", "")
			assert.Equal(t, tc.status, w.Code)
			assert.Contains(t, w.Body.String(), tc.err.Error())
		})
	}

	log := &spyLog{}
	for _, path := range []string{"/signup", "/confirm", "/forgot", "/reset", "/accept"} {
		w := do(&fakeService{err: errors.New("db")}, log, http.MethodPost, path, `{}`)
		assert.Equal(t, http.StatusInternalServerError, w.Code, path)
		assert.Contains(t, w.Body.String(), "erro interno")
	}
	assert.Len(t, log.msgs, 5)
}
