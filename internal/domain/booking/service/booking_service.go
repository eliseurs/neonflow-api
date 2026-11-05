package service

import (
	"context"

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/domain/fleet"
)

type BookingService struct {
	repo booking.Repository
}

func NewBookingService(repo booking.Repository) *BookingService {
	return &BookingService{repo: repo}
}

func (s *BookingService) List(ctx context.Context, vehicleID string) ([]booking.Booking, error) {
	return s.repo.List(ctx, vehicleID)
}

func (s *BookingService) Get(ctx context.Context, id string) (booking.Booking, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *BookingService) Events(ctx context.Context, id string) ([]booking.Event, error) {
	return s.repo.Events(ctx, id)
}

// Create abre a pré-reserva em "preparacao", já bloqueando o período (RN-02.1).
func (s *BookingService) Create(ctx context.Context, req booking.CreateRequest, userID string) (booking.Booking, error) {
	n, err := req.Validate(userID)
	if err != nil {
		return booking.Booking{}, err
	}
	return s.repo.Create(ctx, n)
}

func (s *BookingService) SetChecks(ctx context.Context, id string, checks map[string]bool) (booking.Booking, error) {
	b, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return booking.Booking{}, err
	}
	if err := booking.ValidateChecks(b.Stage, checks); err != nil {
		return booking.Booking{}, err
	}
	return s.repo.SetChecks(ctx, id, b.Stage, checks)
}

// Advance leva a reserva à etapa seguinte quando as conferências da atual estão feitas (RN-02.4).
func (s *BookingService) Advance(ctx context.Context, id, userID string) (booking.Booking, error) {
	b, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return booking.Booking{}, err
	}
	next, err := booking.NextStage(b.Stage, b.Checks)
	if err != nil {
		return booking.Booking{}, err
	}
	return s.repo.Transition(ctx, id, b.Stage, next, userID)
}

// Cancel libera o período (RN-02.5). Multa e estorno (RN-02.21, RN-03.3) ainda não são calculados aqui.
func (s *BookingService) Cancel(ctx context.Context, id, userID string) (booking.Booking, error) {
	b, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return booking.Booking{}, err
	}
	if !booking.Cancelable(b.Stage) {
		return booking.Booking{}, booking.ErrNotCancelable
	}
	return s.repo.Transition(ctx, id, b.Stage, booking.StageCancelada, userID)
}

// Available lista os veículos fora de manutenção e sem reserva ativa no período.
func (s *BookingService) Available(ctx context.Context, start, end string) ([]fleet.Vehicle, error) {
	p, err := booking.ParsePeriod(start, end)
	if err != nil {
		return nil, err
	}
	return s.repo.Available(ctx, p)
}
