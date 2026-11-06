package payout

import "context"

type Repository interface {
	List(ctx context.Context, f Filter) ([]Payout, error)
	// MarkPaid grava o pagamento; com percentual a definir, copia o atual do parceiro.
	MarkPaid(ctx context.Context, id, userID string) (Payout, error)
}

type Service interface {
	List(ctx context.Context, f Filter) ([]Payout, error)
	MarkPaid(ctx context.Context, id, userID string) (Payout, error)
}

type Logger interface {
	Error(msg string, args ...any)
}
