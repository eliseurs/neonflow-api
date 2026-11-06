package terms

import (
	"context"
	"time"
)

type Repository interface {
	Versions(ctx context.Context) ([]Version, error)
	Current(ctx context.Context) (Version, error)
	CreateVersion(ctx context.Context, body, userID string) (Version, error)
	BookingStage(ctx context.Context, bookingID string) (string, error)
	CreateLink(ctx context.Context, bookingID, tokenHash string, expiresAt time.Time, userID string) error
	LinkByHash(ctx context.Context, hash string) (LinkInfo, error)
	Public(ctx context.Context, link LinkInfo) (Public, error)
	// Accept grava o aceite e marca a conferência "terms" da reserva, se a versão ainda for a atual.
	Accept(ctx context.Context, link LinkInfo, version int, typedName string) (Acceptance, error)
	LastAcceptance(ctx context.Context, bookingID string) (*Acceptance, error)
}

type Service interface {
	Versions(ctx context.Context) ([]Version, error)
	Publish(ctx context.Context, body, userID string) (Version, error)
	CreateLink(ctx context.Context, bookingID, userID string) (Link, error)
	BookingAcceptance(ctx context.Context, bookingID string) (*Acceptance, error)
	Public(ctx context.Context, token string) (Public, error)
	Accept(ctx context.Context, token string, req AcceptRequest) (Acceptance, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
