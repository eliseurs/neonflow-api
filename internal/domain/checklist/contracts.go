package checklist

import "context"

type Repository interface {
	List(ctx context.Context) ([]Item, error)
	Create(ctx context.Context, key string, req ItemRequest) (Item, error)
	Update(ctx context.Context, id string, req ItemRequest) (Item, error)
}

type Service interface {
	List(ctx context.Context) ([]Item, error)
	Create(ctx context.Context, req ItemRequest) (Item, error)
	Update(ctx context.Context, id string, req ItemRequest) (Item, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
