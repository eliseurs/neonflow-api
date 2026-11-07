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
	// PublicFleet lista os carros fora de manutenção com a foto de capa, para o site.
	PublicFleet(ctx context.Context) ([]Car, error)
	// PublicCar é a página do carro: ficha e todas as fotos; carro em manutenção não aparece.
	PublicCar(ctx context.Context, id string) (Car, error)
	// VehiclePhoto acha só foto de veículo: foto de reserva nunca é pública.
	VehiclePhoto(ctx context.Context, id string) (Photo, error)
}

// Store é o bucket (internal/infra/storage).
type Store interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) ([]byte, error)
	URL(ctx context.Context, key string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

type Service interface {
	Upload(ctx context.Context, ownerType, ownerID, stage, userID string, body io.Reader, size int64) (Photo, error)
	List(ctx context.Context, ownerType, ownerID, stage string) ([]Photo, error)
	Delete(ctx context.Context, id string) error
	SetCover(ctx context.Context, id string) (Photo, error)
	PublicFleet(ctx context.Context) ([]Car, error)
	PublicCar(ctx context.Context, id string) (Car, error)
	// Variant devolve a versão redimensionada da foto do veículo, gerando e guardando na primeira vez.
	Variant(ctx context.Context, id string, width int) ([]byte, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
