package fleet

import (
	"context"
	"time"
)

type Repository interface {
	List(ctx context.Context) ([]Vehicle, error)
	FindByID(ctx context.Context, id string) (Vehicle, error)
	Create(ctx context.Context, v VehicleRequest) (Vehicle, error)
	Update(ctx context.Context, id string, v VehicleRequest) (Vehicle, error)
	// Stats do período [since, agora): de um carro (id) ou de todos (id vazio).
	Stats(ctx context.Context, id string, since time.Time) ([]Stats, error)
}

type Service interface {
	List(ctx context.Context) ([]Vehicle, error)
	Get(ctx context.Context, id string) (Vehicle, error)
	Create(ctx context.Context, req VehicleRequest) (Vehicle, error)
	Update(ctx context.Context, id string, req VehicleRequest) (Vehicle, error)
	// Comparison é o desempenho de toda a frota nos últimos 12 meses.
	Comparison(ctx context.Context) ([]Stats, error)
	VehicleStats(ctx context.Context, id string) (Stats, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
