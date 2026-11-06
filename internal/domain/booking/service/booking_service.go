package service

import (
	"context"
	"time"

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/domain/fleet"
)

type BookingService struct {
	repo booking.Repository
	now  func() time.Time
}

func NewBookingService(repo booking.Repository) *BookingService {
	return &BookingService{repo: repo, now: time.Now}
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
	extra, err := s.extraChecks(ctx, b.Stage)
	if err != nil {
		return booking.Booking{}, err
	}
	if err := booking.ValidateChecks(b.Stage, checks, extra...); err != nil {
		return booking.Booking{}, err
	}
	return s.repo.SetChecks(ctx, id, b.Stage, checks)
}

// Advance leva a reserva à etapa seguinte quando as conferências da atual estão feitas (RN-02.4).
// Na entrega e na devolução, grava a hora real (RN-02.19); na devolução, calcula a diária extra do atraso.
func (s *BookingService) Advance(ctx context.Context, id, userID string, req booking.AdvanceRequest) (booking.Booking, error) {
	b, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return booking.Booking{}, err
	}
	extra, err := s.extraChecks(ctx, b.Stage)
	if err != nil {
		return booking.Booking{}, err
	}
	next, err := booking.NextStage(b.Stage, b.Checks, extra...)
	if err != nil {
		return booking.Booking{}, err
	}
	rec, err := s.record(b, next, req.At)
	if err != nil {
		return booking.Booking{}, err
	}
	return s.repo.Transition(ctx, id, b.Stage, next, userID, rec)
}

// record monta o que a transição grava: a hora real na entrega e na devolução e, na devolução, o atraso.
func (s *BookingService) record(b booking.Booking, next, at string) (booking.Record, error) {
	if next != booking.StageEmLocacao && next != booking.StageDevolvida {
		return booking.Record{}, nil
	}
	now := s.now()
	when := now
	if at != "" {
		t, err := booking.ParseInstant(at)
		if err != nil {
			return booking.Record{}, err
		}
		if t.After(now.Add(booking.ClockSkew)) {
			return booking.Record{}, booking.ErrFutureTime
		}
		when = t
	}
	if next == booking.StageEmLocacao {
		return booking.Record{DeliveredAt: &when}, nil
	}
	if b.DeliveredAt != nil && !when.After(*b.DeliveredAt) {
		return booking.Record{}, booking.ErrReturnBeforeDelivery
	}
	days := booking.LateDays(b.EndAt, when)
	cents := int64(days) * b.DailyRateCents
	return booking.Record{ReturnedAt: &when, ExtraDays: &days, ExtraCents: &cents}, nil
}

// extraChecks: na etapa "devolvida", o checklist de devolução configurado pela central entra
// nas conferências exigidas para encerrar (RN-01.6).
func (s *BookingService) extraChecks(ctx context.Context, stage string) ([]string, error) {
	if stage != booking.StageDevolvida {
		return nil, nil
	}
	return s.repo.ReturnChecklist(ctx)
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
	return s.repo.Transition(ctx, id, b.Stage, booking.StageCancelada, userID, booking.Record{})
}

// Available lista os veículos fora de manutenção e sem reserva ativa no período.
func (s *BookingService) Available(ctx context.Context, start, end string) ([]fleet.Vehicle, error) {
	p, err := booking.ParsePeriod(start, end)
	if err != nil {
		return nil, err
	}
	return s.repo.Available(ctx, p)
}
