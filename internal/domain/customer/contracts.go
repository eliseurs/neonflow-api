package customer

import "context"

type Repository interface {
	// List busca por nome, WhatsApp ou e-mail (vazio = todos), os mais recentes primeiro.
	List(ctx context.Context, text, digits string) ([]Customer, error)
	Get(ctx context.Context, id string) (Detail, error)
	Create(ctx context.Context, r Request) (Customer, error)
	Update(ctx context.Context, id string, r Request) (Customer, error)
	// Anonymize apaga nome e contatos e mantém os pedidos (LGPD art. 18).
	Anonymize(ctx context.Context, id string) (Customer, error)
}

type Service interface {
	List(ctx context.Context, q string) ([]Customer, error)
	Get(ctx context.Context, id string) (Detail, error)
	Create(ctx context.Context, r Request) (Customer, error)
	Update(ctx context.Context, id string, r Request) (Customer, error)
	Anonymize(ctx context.Context, id string) (Customer, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
