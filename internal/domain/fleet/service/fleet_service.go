package service

import (
	"context"
	"time"

	"neonflow-api/internal/domain/fleet"
)

type FleetService struct {
	repo fleet.Repository
	now  func() time.Time
}

func NewFleetService(repo fleet.Repository) *FleetService {
	return &FleetService{repo: repo, now: time.Now}
}

func (s *FleetService) Comparison(ctx context.Context) ([]fleet.Stats, error) {
	return s.repo.Stats(ctx, "", s.now().Add(-fleet.StatsWindow))
}

func (s *FleetService) VehicleStats(ctx context.Context, id string) (fleet.Stats, error) {
	list, err := s.repo.Stats(ctx, id, s.now().Add(-fleet.StatsWindow))
	if err != nil {
		return fleet.Stats{}, err
	}
	if len(list) == 0 {
		return fleet.Stats{}, fleet.ErrVehicleNotFound
	}
	return list[0], nil
}

func (s *FleetService) List(ctx context.Context) ([]fleet.Vehicle, error) {
	return s.repo.List(ctx)
}

func (s *FleetService) Get(ctx context.Context, id string) (fleet.Vehicle, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *FleetService) Create(ctx context.Context, req fleet.VehicleRequest) (fleet.Vehicle, error) {
	valid, err := req.Validate()
	if err != nil {
		return fleet.Vehicle{}, err
	}
	return s.repo.Create(ctx, valid)
}

// Update substitui todos os campos. Mudar a diária não altera reservas já feitas (RN-02.6).
func (s *FleetService) Update(ctx context.Context, id string, req fleet.VehicleRequest) (fleet.Vehicle, error) {
	valid, err := req.Validate()
	if err != nil {
		return fleet.Vehicle{}, err
	}
	return s.repo.Update(ctx, id, valid)
}
