package user

import (
	"context"

	"neonflow-api/internal/domain/authentication"
)

type Repository interface {
	List(ctx context.Context) ([]User, error)
	FindByID(ctx context.Context, id string) (User, error)
	Create(ctx context.Context, u authentication.NewUser) (User, error)
	Update(ctx context.Context, id string, req UpdateRequest) (User, error)
	PasswordHash(ctx context.Context, id string) (string, error)
	SetPassword(ctx context.Context, id, hash string) error
}

type Service interface {
	List(ctx context.Context) ([]User, error)
	Create(ctx context.Context, req CreateRequest) (User, error)
	Update(ctx context.Context, actorID, id string, req UpdateRequest) (User, error)
	ResetPassword(ctx context.Context, id string, req PasswordRequest) error
	ChangeOwnPassword(ctx context.Context, actorID string, req ChangeOwnPasswordRequest) error
}

type Logger interface {
	Error(msg string, args ...any)
}
