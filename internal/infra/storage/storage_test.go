package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/config"
)

func TestNew(t *testing.T) {
	s, err := New(config.Storage{})
	require.NoError(t, err)
	assert.IsType(t, Disabled{}, s, "sem variáveis, desligado")

	_, err = New(config.Storage{Endpoint: "::", Bucket: "b", AccessKey: "a", SecretKey: "s"})
	assert.Error(t, err)

	s, err = New(config.Storage{Endpoint: "https://t3.storageapi.dev", Region: "auto", Bucket: "neonflow-teste", AccessKey: "a", SecretKey: "s"})
	require.NoError(t, err)
	b, ok := s.(*Bucket)
	require.True(t, ok)
	// Assinar não acessa a rede: o link sai com a chave e o prazo.
	u, err := b.URL(context.Background(), "veiculos/v1/foto.jpg", 15*time.Minute)
	require.NoError(t, err)
	assert.Contains(t, u, "veiculos/v1/foto.jpg")
	assert.Contains(t, u, "X-Amz-Expires=900")
}

func TestDisabled(t *testing.T) {
	ctx := context.Background()
	d := Disabled{}
	assert.ErrorIs(t, d.Put(ctx, "k", strings.NewReader("x"), 1, "image/png"), ErrNotConfigured)
	_, err := d.URL(ctx, "k", time.Minute)
	assert.ErrorIs(t, err, ErrNotConfigured)
	assert.ErrorIs(t, d.Delete(ctx, "k"), ErrNotConfigured)
}

func TestCleanKey(t *testing.T) {
	assert.Equal(t, "veiculos/v1/a.jpg", CleanKey("veiculos", "/v1/", "a.jpg"))
	assert.Equal(t, "reservas/b1/devolucao/a.jpg", CleanKey("reservas", "../b1", "devolucao", "a.jpg"))
	assert.Equal(t, "a", CleanKey("", "a", ""))
}

func TestBucketPutDelete(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.Path)
		if strings.Contains(r.URL.Path, "erro") {
			w.WriteHeader(http.StatusForbidden) // 403 não é repetido pelo cliente
			return
		}
		if r.Method == http.MethodPut {
			w.Header().Set("ETag", `"abc"`)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	s, err := New(config.Storage{Endpoint: srv.URL, Region: "auto", Bucket: "neonflow-teste", AccessKey: "a", SecretKey: "s"})
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, s.Put(ctx, "veiculos/v1/a.jpg", strings.NewReader("img"), 3, "image/jpeg"))
	require.NoError(t, s.Delete(ctx, "veiculos/v1/a.jpg"))
	assert.Contains(t, got, "PUT /neonflow-teste/veiculos/v1/a.jpg")
	assert.Contains(t, got, "DELETE /neonflow-teste/veiculos/v1/a.jpg")
	assert.Error(t, s.Put(ctx, "erro.jpg", strings.NewReader("img"), 3, "image/jpeg"))
	assert.Error(t, s.Delete(ctx, "erro.jpg"))
}
