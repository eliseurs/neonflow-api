package service

import (
	"context"

	"neonflow-api/internal/domain/customer"
)

type CustomerService struct {
	repo customer.Repository
}

func NewCustomerService(repo customer.Repository) *CustomerService {
	return &CustomerService{repo: repo}
}

func (s *CustomerService) List(ctx context.Context, q string) ([]customer.Customer, error) {
	text, digits := customer.NormalizeQuery(q)
	return s.repo.List(ctx, text, digits)
}

func (s *CustomerService) Get(ctx context.Context, id string) (customer.Detail, error) {
	return s.repo.Get(ctx, id)
}

func (s *CustomerService) Create(ctx context.Context, r customer.Request) (customer.Customer, error) {
	valid, err := r.Validate()
	if err != nil {
		return customer.Customer{}, err
	}
	return s.repo.Create(ctx, valid)
}

func (s *CustomerService) Update(ctx context.Context, id string, r customer.Request) (customer.Customer, error) {
	valid, err := r.Validate()
	if err != nil {
		return customer.Customer{}, err
	}
	return s.repo.Update(ctx, id, valid)
}

func (s *CustomerService) Anonymize(ctx context.Context, id string) (customer.Customer, error) {
	return s.repo.Anonymize(ctx, id)
}
