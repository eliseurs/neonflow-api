package service

import (
	"context"

	"neonflow-api/internal/domain/partnerarea"
)

type PartnerAreaService struct {
	repo partnerarea.Repository
}

func NewPartnerAreaService(repo partnerarea.Repository) *PartnerAreaService {
	return &PartnerAreaService{repo: repo}
}

// Area monta a área pelo parceiro da conta: carros e reservas só para o proprietário, corridas só para o
// motorista; indicações para todos, porque todo parceiro pode indicar (RN-07.2).
func (s *PartnerAreaService) Area(ctx context.Context, userID string) (partnerarea.Area, error) {
	p, err := s.repo.Partner(ctx, userID)
	if err != nil {
		return partnerarea.Area{}, err
	}
	a := partnerarea.Area{Partner: p, Vehicles: []partnerarea.Vehicle{}, Bookings: []partnerarea.Booking{}, Rides: []partnerarea.Ride{}}
	if p.Has(partnerarea.KindOwner) {
		if a.Vehicles, err = s.repo.Vehicles(ctx, p.ID); err != nil {
			return a, err
		}
		if a.Bookings, err = s.repo.Bookings(ctx, p.ID); err != nil {
			return a, err
		}
	}
	if p.Has(partnerarea.KindDriver) {
		if a.Rides, err = s.repo.Rides(ctx, p.ID); err != nil {
			return a, err
		}
	}
	a.Referrals, err = s.repo.Referrals(ctx, p.ID)
	return a, err
}
