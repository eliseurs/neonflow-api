// Package storage guarda arquivos num bucket compatível com S3: o bucket do Railway hoje, o Cloudflare R2
// depois, trocando só as variáveis (ADR-0014). O bucket é privado: leitura só por link assinado e temporário.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"neonflow-api/internal/config"
)

var (
	// ErrNotConfigured: sem as variáveis STORAGE_*, não há onde guardar arquivos.
	ErrNotConfigured = errors.New("armazenamento de arquivos não configurado")
	// ErrNotFound: a chave não existe no bucket.
	ErrNotFound = errors.New("arquivo não encontrado")
)

// Store é o que os domínios usam; os testes usam um falso.
type Store interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	// Get lê o arquivo inteiro (fotos de até 10 MB); ErrNotFound se a chave não existe.
	Get(ctx context.Context, key string) ([]byte, error)
	URL(ctx context.Context, key string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

// New devolve o bucket configurado ou um Store que sempre responde ErrNotConfigured.
func New(cfg config.Storage) (Store, error) {
	if !cfg.Enabled() {
		return Disabled{}, nil
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("storage endpoint inválido: %q", cfg.Endpoint)
	}
	client, err := minio.New(u.Host, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       u.Scheme != "http",
		Region:       cfg.Region,
		BucketLookup: minio.BucketLookupAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("storage client: %w", err)
	}
	return &Bucket{client: client, bucket: cfg.Bucket}, nil
}

// Bucket é o Store sobre o minio-go.
type Bucket struct {
	client *minio.Client
	bucket string
}

func (b *Bucket) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := b.client.PutObject(ctx, b.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (b *Bucket) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := b.client.GetObject(ctx, b.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	defer obj.Close() //nolint:errcheck
	data, err := io.ReadAll(obj)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read object: %w", err)
	}
	return data, nil
}

func (b *Bucket) URL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := b.client.PresignedGetObject(ctx, b.bucket, key, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("presign: %w", err)
	}
	return u.String(), nil
}

func (b *Bucket) Delete(ctx context.Context, key string) error {
	if err := b.client.RemoveObject(ctx, b.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}

// Disabled é o Store sem configuração.
type Disabled struct{}

func (Disabled) Put(context.Context, string, io.Reader, int64, string) error { return ErrNotConfigured }
func (Disabled) Get(context.Context, string) ([]byte, error)                 { return nil, ErrNotConfigured }
func (Disabled) URL(context.Context, string, time.Duration) (string, error) {
	return "", ErrNotConfigured
}
func (Disabled) Delete(context.Context, string) error { return ErrNotConfigured }

// CleanKey impede caminhos fora da pasta do dono ("..", barras duplicadas).
func CleanKey(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(strings.ReplaceAll(p, "..", ""), "/")
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
}
