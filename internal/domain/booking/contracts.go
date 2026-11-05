package booking

import (
	"context"

	"neonflow-api/internal/domain/fleet"
)

type Repository interface {
	List(ctx context.Context, vehicleID string) ([]Booking, error)
	FindByID(ctx context.Context, id string) (Booking, error)
	Events(ctx context.Context, id string) ([]Event, error)
	Create(ctx context.Context, n NewBooking) (Booking, error)
	SetChecks(ctx context.Context, id, stage string, checks map[string]bool) (Booking, error)
	Transition(ctx context.Context, id, from, to, userID string) (Booking, error)
	Available(ctx context.Context, p Period) ([]fleet.Vehicle, error)
}

type Service interface {
	List(ctx context.Context, vehicleID string) ([]Booking, error)
	Get(ctx context.Context, id string) (Booking, error)
	Events(ctx context.Context, id string) ([]Event, error)
	Create(ctx context.Context, req CreateRequest, userID string) (Booking, error)
	SetChecks(ctx context.Context, id string, checks map[string]bool) (Booking, error)
	Advance(ctx context.Context, id, userID string) (Booking, error)
	Cancel(ctx context.Context, id, userID string) (Booking, error)
	Available(ctx context.Context, start, end string) ([]fleet.Vehicle, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
