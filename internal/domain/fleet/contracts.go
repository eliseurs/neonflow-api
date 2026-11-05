package fleet

import "context"

type Repository interface {
	List(ctx context.Context) ([]Vehicle, error)
	FindByID(ctx context.Context, id string) (Vehicle, error)
	Create(ctx context.Context, v VehicleRequest) (Vehicle, error)
	Update(ctx context.Context, id string, v VehicleRequest) (Vehicle, error)
}

type Service interface {
	List(ctx context.Context) ([]Vehicle, error)
	Get(ctx context.Context, id string) (Vehicle, error)
	Create(ctx context.Context, req VehicleRequest) (Vehicle, error)
	Update(ctx context.Context, id string, req VehicleRequest) (Vehicle, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
