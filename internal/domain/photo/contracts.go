package photo

import (
	"context"
	"io"
	"time"
)

type Repository interface {
	// BookingStage devolve a etapa da reserva; ErrOwnerNotFound se não existe.
	BookingStage(ctx context.Context, id string) (string, error)
	VehicleExists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, n NewPhoto) (Photo, error)
	List(ctx context.Context, ownerType, ownerID, stage string) ([]Photo, error)
	Delete(ctx context.Context, id string) (Photo, error)
	// SetCover põe a foto do veículo em primeiro lugar.
	SetCover(ctx context.Context, id string) (Photo, error)
}

// Store é o bucket (internal/infra/storage).
type Store interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	URL(ctx context.Context, key string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

type Service interface {
	Upload(ctx context.Context, ownerType, ownerID, stage, userID string, body io.Reader, size int64) (Photo, error)
	List(ctx context.Context, ownerType, ownerID, stage string) ([]Photo, error)
	Delete(ctx context.Context, id string) error
	SetCover(ctx context.Context, id string) (Photo, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
