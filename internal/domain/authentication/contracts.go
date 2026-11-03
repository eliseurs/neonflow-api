package authentication

import (
	"context"
	"time"
)

type Repository interface {
	FindByEmail(ctx context.Context, email string) (Credentials, error)
	FindByID(ctx context.Context, id string) (User, error)
	Create(ctx context.Context, u NewUser) (User, error)
}

type Service interface {
	Login(ctx context.Context, req LoginRequest) (LoginResponse, error)
	Me(ctx context.Context, userID string) (User, error)
	CreateUser(ctx context.Context, req CreateUserRequest) (User, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

type TokenIssuer interface {
	Generate(userID, role string) (string, time.Time, error)
}

type Logger interface {
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
