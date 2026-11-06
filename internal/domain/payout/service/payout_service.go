package service

import (
	"context"

	"neonflow-api/internal/domain/payout"
)

type PayoutService struct {
	repo payout.Repository
}

func NewPayoutService(repo payout.Repository) *PayoutService {
	return &PayoutService{repo: repo}
}

func (s *PayoutService) List(ctx context.Context, f payout.Filter) ([]payout.Payout, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// MarkPaid registra que a central pagou o parceiro; o pagamento é feito fora do sistema.
func (s *PayoutService) MarkPaid(ctx context.Context, id, userID string) (payout.Payout, error) {
	return s.repo.MarkPaid(ctx, id, userID)
}
