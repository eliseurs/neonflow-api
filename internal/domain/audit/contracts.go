package audit

import "context"

type Repository interface {
	List(ctx context.Context, f Filter) ([]Entry, error)
}

type Service interface {
	List(ctx context.Context, f Filter) ([]Entry, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
