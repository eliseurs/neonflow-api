package contract

import "context"

type Repository interface {
	Documents(ctx context.Context) ([]Document, error)
	Versions(ctx context.Context, code string) ([]Summary, error)
	CreateVersion(ctx context.Context, code string, req VersionRequest, sha string, actor Actor) (Summary, error)
	Version(ctx context.Context, id string) (Version, error)
	AddComment(ctx context.Context, versionID string, req CommentRequest, actor Actor) (Comment, error)
	// AddEvent trava a versão, lê a situação atual, chama next e grava o evento numa transação.
	AddEvent(ctx context.Context, versionID, action, note string, actor Actor,
		next func(current, sha string) (string, error)) (Event, error)
}

type Service interface {
	Documents(ctx context.Context) ([]Document, error)
	Versions(ctx context.Context, code string) ([]Summary, error)
	CreateVersion(ctx context.Context, code string, req VersionRequest, actor Actor) (Summary, error)
	Version(ctx context.Context, id string) (Version, error)
	AddComment(ctx context.Context, versionID string, req CommentRequest, actor Actor) (Comment, error)
	AddEvent(ctx context.Context, versionID string, req EventRequest, actor Actor) (Event, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
