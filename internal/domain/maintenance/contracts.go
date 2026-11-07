package maintenance

import "context"

type Repository interface {
	List(ctx context.Context, vehicleID string) ([]Record, error)
	Create(ctx context.Context, vehicleID, userID string, r Request) (Record, error)
	Update(ctx context.Context, id string, r Request) (Record, error)
	Delete(ctx context.Context, id string) error
}

type Service interface {
	List(ctx context.Context, vehicleID string) ([]Record, error)
	Create(ctx context.Context, vehicleID, userID string, r Request) (Record, error)
	Update(ctx context.Context, id string, r Request) (Record, error)
	Delete(ctx context.Context, id string) error
}

type Logger interface {
	Error(msg string, args ...any)
}
