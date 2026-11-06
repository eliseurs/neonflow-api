package dashboard

import (
	"context"
	"time"
)

type Repository interface {
	Totals(ctx context.Context, start, end time.Time) (Totals, error)
}

type Service interface {
	Summary(ctx context.Context, month string) (Summary, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
