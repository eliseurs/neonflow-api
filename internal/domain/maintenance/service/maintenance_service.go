package service

import (
	"context"
	"time"

	"neonflow-api/internal/domain/maintenance"
)

type MaintenanceService struct {
	repo maintenance.Repository
	now  func() time.Time
}

func NewMaintenanceService(repo maintenance.Repository) *MaintenanceService {
	return &MaintenanceService{repo: repo, now: time.Now}
}

func (s *MaintenanceService) List(ctx context.Context, vehicleID string) ([]maintenance.Record, error) {
	return s.repo.List(ctx, vehicleID)
}

func (s *MaintenanceService) Create(ctx context.Context, vehicleID, userID string, r maintenance.Request) (maintenance.Record, error) {
	valid, err := r.Validate(s.now())
	if err != nil {
		return maintenance.Record{}, err
	}
	return s.repo.Create(ctx, vehicleID, userID, valid)
}

func (s *MaintenanceService) Update(ctx context.Context, id string, r maintenance.Request) (maintenance.Record, error) {
	valid, err := r.Validate(s.now())
	if err != nil {
		return maintenance.Record{}, err
	}
	return s.repo.Update(ctx, id, valid)
}

func (s *MaintenanceService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
