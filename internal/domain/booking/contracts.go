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
	// Transition muda a etapa e grava o que Record trouxer (hora real e atraso).
	Transition(ctx context.Context, id, from, to, userID string, rec Record) (Booking, error)
	Available(ctx context.Context, p Period) ([]fleet.Vehicle, error)
	// ReturnChecklist devolve as chaves dos itens ativos do checklist de devolução, em ordem (RN-01.6).
	ReturnChecklist(ctx context.Context) ([]string, error)
}

type Service interface {
	List(ctx context.Context, vehicleID string) ([]Booking, error)
	Get(ctx context.Context, id string) (Booking, error)
	Events(ctx context.Context, id string) ([]Event, error)
	Create(ctx context.Context, req CreateRequest, userID string) (Booking, error)
	SetChecks(ctx context.Context, id string, checks map[string]bool) (Booking, error)
	Advance(ctx context.Context, id, userID string, req AdvanceRequest) (Booking, error)
	Cancel(ctx context.Context, id, userID string) (Booking, error)
	Available(ctx context.Context, start, end string) ([]fleet.Vehicle, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
