package transfer

import "context"

type Repository interface {
	List(ctx context.Context) ([]Transfer, error)
	FindByID(ctx context.Context, id string) (Transfer, error)
	Events(ctx context.Context, id string) ([]Event, error)
	Create(ctx context.Context, n NewTransfer) (Transfer, error)
	Drivers(ctx context.Context, id string) ([]Driver, error)
	Offer(ctx context.Context, id, userID string, offerCents int64) (Transfer, error)
	Notify(ctx context.Context, id, userID, driverID string) (Transfer, error)
	Accept(ctx context.Context, id, userID string, req AcceptRequest) (Transfer, error)
	Release(ctx context.Context, id, userID string) (Transfer, error)
	SetPaid(ctx context.Context, id, userID string, paid bool) (Transfer, error)
	Start(ctx context.Context, id, userID string) (Transfer, error)
	Complete(ctx context.Context, id, userID string, extraWaitCents int64) (Transfer, error)
	Cancel(ctx context.Context, id, userID string, feeCents int64, flightCancelled bool) (Transfer, error)
}

type Service interface {
	List(ctx context.Context) ([]Transfer, error)
	Get(ctx context.Context, id string) (Transfer, error)
	Events(ctx context.Context, id string) ([]Event, error)
	Create(ctx context.Context, req CreateRequest, userID string) (Transfer, error)
	Drivers(ctx context.Context, id string) ([]Driver, error)
	Offer(ctx context.Context, id, userID string, req OfferRequest) (Transfer, error)
	Notify(ctx context.Context, id, userID string, req NotifyRequest) (Transfer, error)
	Accept(ctx context.Context, id, userID string, req AcceptRequest) (Transfer, error)
	Release(ctx context.Context, id, userID string) (Transfer, error)
	SetPaid(ctx context.Context, id, userID string, paid bool) (Transfer, error)
	Start(ctx context.Context, id, userID string) (Transfer, error)
	Complete(ctx context.Context, id, userID string, req CompleteRequest) (Transfer, error)
	Cancel(ctx context.Context, id, userID string, req CancelRequest) (Transfer, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
