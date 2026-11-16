package service

import (
	"context"
	"errors"
	"strings"

	"neonflow-api/internal/domain/partnerarea"
	"neonflow-api/internal/domain/transfer"
)

type PartnerAreaService struct {
	repo      partnerarea.Repository
	transfers partnerarea.Transfers
}

func NewPartnerAreaService(repo partnerarea.Repository, transfers partnerarea.Transfers) *PartnerAreaService {
	return &PartnerAreaService{repo: repo, transfers: transfers}
}

// Area monta a área pelo parceiro da conta: carros e reservas só para o proprietário, corridas e ofertas
// abertas só para o motorista; indicações para todos, porque todo parceiro pode indicar (RN-07.2).
func (s *PartnerAreaService) Area(ctx context.Context, userID string) (partnerarea.Area, error) {
	p, err := s.repo.Partner(ctx, userID)
	if err != nil {
		return partnerarea.Area{}, err
	}
	a := partnerarea.Area{Partner: p, Vehicles: []partnerarea.Vehicle{}, Bookings: []partnerarea.Booking{}, Rides: []partnerarea.Ride{}, Offers: []partnerarea.Offer{}}
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
		if a.Offers, err = s.repo.Offers(ctx); err != nil {
			return a, err
		}
	}
	a.Referrals, err = s.repo.Referrals(ctx, p.ID)
	return a, err
}

// Accept: o motorista aceita a corrida da oferta aberta (PEN-74). O transfer trava a corrida, então o
// primeiro aceite leva e o segundo recebe ErrWrongStage. O carro, se vier, é um dos dele.
func (s *PartnerAreaService) Accept(ctx context.Context, userID, transferID string, req partnerarea.AcceptRequest) error {
	p, err := s.driver(ctx, userID)
	if err != nil {
		return err
	}
	vehicle := strings.TrimSpace(req.VehicleID)
	if vehicle != "" {
		owns, err := s.repo.OwnsVehicle(ctx, p.ID, vehicle)
		if err != nil {
			return err
		}
		if !owns {
			return partnerarea.ErrNotYourVehicle
		}
	}
	_, err = s.transfers.Accept(ctx, transferID, userID, transfer.AcceptRequest{DriverPartnerID: p.ID, VehicleID: vehicle})
	return err
}

// Release: o motorista desiste da corrida que aceitou, antes de começar; ela volta para a oferta com o mesmo
// valor, sem penalidade (RN-02.31).
func (s *PartnerAreaService) Release(ctx context.Context, userID, transferID string) error {
	p, err := s.driver(ctx, userID)
	if err != nil {
		return err
	}
	t, err := s.transfers.Get(ctx, transferID)
	if errors.Is(err, transfer.ErrNotFound) {
		return partnerarea.ErrNotYourRide
	}
	if err != nil {
		return err
	}
	if t.DriverPartnerID == nil || *t.DriverPartnerID != p.ID {
		return partnerarea.ErrNotYourRide
	}
	_, err = s.transfers.Release(ctx, transferID, userID)
	return err
}

func (s *PartnerAreaService) driver(ctx context.Context, userID string) (partnerarea.Partner, error) {
	p, err := s.repo.Partner(ctx, userID)
	if err != nil {
		return p, err
	}
	if !p.Has(partnerarea.KindDriver) {
		return p, partnerarea.ErrNotDriver
	}
	return p, nil
}
