package partner

import "context"

type Repository interface {
	List(ctx context.Context, kind string) ([]Partner, error)
	FindByID(ctx context.Context, id string) (Partner, error)
	FindByCode(ctx context.Context, code string) (Partner, error)
	Create(ctx context.Context, code string, req Request) (Partner, error)
	Update(ctx context.Context, id string, req Request) (Partner, error)
}

type Service interface {
	List(ctx context.Context, kind string) ([]Partner, error)
	Get(ctx context.Context, id string) (Partner, error)
	ByCode(ctx context.Context, code string) (Partner, error)
	Create(ctx context.Context, req Request) (Partner, error)
	Update(ctx context.Context, id string, req Request) (Partner, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
