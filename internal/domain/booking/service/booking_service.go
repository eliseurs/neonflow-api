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
	// "Fotos obrigatórias" (q80): o item só é marcado com ao menos uma foto da devolução enviada.
	if checks[booking.PhotosCheck] && b.Stage == booking.StageDevolvida {
		n, err := s.repo.ReturnPhotos(ctx, id)
		if err != nil {
			return booking.Booking{}, err
		}
		if n == 0 {
			return booking.Booking{}, booking.ErrPhotosRequired
		}
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
	rec, err := s.record(b, next, req)
	if err != nil {
		return booking.Booking{}, err
	}
	return s.repo.Transition(ctx, id, b.Stage, next, userID, rec)
}

// record monta o que a transição grava: a hora real na entrega e na devolução e, na devolução, o atraso.
func (s *BookingService) record(b booking.Booking, next string, req booking.AdvanceRequest) (booking.Record, error) {
	at := req.At
	if next != booking.StageEmLocacao && next != booking.StageDevolvida {
		return booking.Record{}, nil
	}
	if req.Km != nil && *req.Km < 0 {
		return booking.Record{}, booking.ErrInvalidKm
	}
	limited := b.KmLimitPerDay != nil
	if limited && req.Km == nil {
		return booking.Record{}, booking.ErrKmRequired
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
		return booking.Record{DeliveredAt: &when, DeliveredKm: req.Km}, nil
	}
	if b.DeliveredAt != nil && !when.After(*b.DeliveredAt) {
		return booking.Record{}, booking.ErrReturnBeforeDelivery
	}
	if req.Km != nil && b.DeliveredKm != nil && *req.Km < *b.DeliveredKm {
		return booking.Record{}, booking.ErrKmBeforeDelivery
	}
	minutes := booking.LateMinutes(b.EndAt, when)
	cents := booking.LateCents(minutes, b.DailyRateCents)
	rec := booking.Record{ReturnedAt: &when, ExtraMinutes: &minutes, ExtraCents: &cents, ReturnedKm: req.Km}
	// Sem o odômetro da entrega (entregue antes desta versão), não há como medir o excedente.
	if limited && b.DeliveredKm != nil {
		excess := booking.ExcessKm(*b.KmLimitPerDay, b.Days, *req.Km-*b.DeliveredKm)
		rec.ExcessKm = &excess
		// Sem valor do km cadastrado, o excedente é medido e o valor fica com a central (RN-02.29).
		if b.ExtraKmCents != nil {
			excessCents := int64(excess) * *b.ExtraKmCents
			rec.ExcessKmCents = &excessCents
		}
	}
	return rec, nil
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
