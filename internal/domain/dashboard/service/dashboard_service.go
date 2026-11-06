package service

import (
	"context"
	"time"

	"neonflow-api/internal/domain/dashboard"
)

type DashboardService struct {
	repo dashboard.Repository
	now  func() time.Time
}

func NewDashboardService(repo dashboard.Repository) *DashboardService {
	return &DashboardService{repo: repo, now: time.Now}
}

// Summary do mês (AAAA-MM, vazio = atual em Gramado); a ocupação é sempre a de agora.
func (s *DashboardService) Summary(ctx context.Context, month string) (dashboard.Summary, error) {
	m, start, end, err := dashboard.MonthRange(month, s.now())
	if err != nil {
		return dashboard.Summary{}, err
	}
	t, err := s.repo.Totals(ctx, start, end)
	if err != nil {
		return dashboard.Summary{}, err
	}
	return dashboard.Summary{
		Month:              m,
		PickupRevenueCents: t.PickupRevenueCents,
		PickupCount:        t.PickupCount,
		PaidCents:          t.PaidCents,
		PaidCount:          t.PaidCount,
		RentedToday:        t.RentedToday,
		FleetSize:          t.FleetSize,
		OccupancyPercent:   dashboard.Occupancy(t.RentedToday, t.FleetSize),
	}, nil
}
