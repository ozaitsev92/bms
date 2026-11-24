package app

import (
	"context"
	"fmt"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

// ApartmentService coordinates apartment-related business operations.
type ApartmentService struct {
	repo port.ApartmentRepository
}

// NewApartmentService creates an apartment application service.
func NewApartmentService(repo port.ApartmentRepository) *ApartmentService {
	return &ApartmentService{repo: repo}
}

// List returns a page of apartments.
func (s *ApartmentService) List(ctx context.Context, p port.ListParams) ([]domain.Apartment, error) {
	apartments, err := s.repo.FindAll(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("list apartments: %w", err)
	}

	return apartments, nil
}

// GetByID returns an apartment by ID.
func (s *ApartmentService) GetByID(ctx context.Context, id int) (*domain.Apartment, error) {
	apartment, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get apartment %d: %w", id, err)
	}

	return apartment, nil
}

// ListByBuilding returns a page of apartments of an existing building.
func (s *ApartmentService) ListByBuilding(ctx context.Context, buildingID int, p port.ListParams) ([]domain.Apartment, error) {
	apartments, err := s.repo.FindByBuildingID(ctx, buildingID, p)
	if err != nil {
		return nil, fmt.Errorf("list apartments of building %d: %w", buildingID, err)
	}

	return apartments, nil
}

// Upsert validates and creates or updates an apartment. created reports
// whether a new apartment was inserted.
func (s *ApartmentService) Upsert(ctx context.Context, a domain.Apartment) (*domain.Apartment, bool, error) {
	if err := a.Validate(); err != nil {
		return nil, false, err
	}

	apartment, created, err := s.repo.Upsert(ctx, a)
	if err != nil {
		return nil, false, fmt.Errorf("upsert apartment: %w", err)
	}

	return apartment, created, nil
}

// Delete removes an apartment by ID.
func (s *ApartmentService) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete apartment %d: %w", id, err)
	}

	return nil
}
