package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/config"
)

func TestNew_Disabled(t *testing.T) {
	s := New(config.Email{APIKey: "re_teste"})
	assert.False(t, s.Enabled())
	assert.ErrorIs(t, s.Send(context.Background(), Message{To: "a@neonflow.test"}), ErrNotConfigured)
}

func TestResend_Send(t *testing.T) {
	var got map[string]any
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/emails", r.URL.Path)
		auth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"email-1"}`))
	}))
	defer srv.Close()

	s := New(config.Email{APIKey: "re_teste", From: "Neon Flow <contato@neonflow.test>"})
	require.True(t, s.Enabled())
	r := s.(*Resend)
	require.NoError(t, r.setBaseURL(srv.URL))

	require.NoError(t, s.Send(context.Background(), Message{To: "ana@neonflow.test", Subject: "Assunto", Text: "Corpo"}))
	assert.Equal(t, "Bearer re_teste", auth)
	assert.Equal(t, "Neon Flow <contato@neonflow.test>", got["from"])
	assert.Equal(t, []any{"ana@neonflow.test"}, got["to"])
	assert.Equal(t, "Assunto", got["subject"])
	assert.Equal(t, "Corpo", got["text"])
}

func TestResend_SendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"statusCode":403,"name":"validation_error","message":"domínio não verificado"}`))
	}))
	defer srv.Close()

	r := New(config.Email{APIKey: "re_teste", From: "contato@neonflow.test"}).(*Resend)
	require.NoError(t, r.setBaseURL(srv.URL))
	err := r.Send(context.Background(), Message{To: "ana@neonflow.test", Subject: "s", Text: "t"})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "ana@neonflow.test", "o erro vai para o log e não leva o destinatário")
}
