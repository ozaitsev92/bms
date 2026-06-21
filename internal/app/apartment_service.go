package app

import (
	"context"
	"database/sql"
	"errors"
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

// GetAll returns all apartments.
func (s *ApartmentService) GetAll(ctx context.Context) ([]domain.Apartment, error) {
	apartments, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all apartments: %w", err)
	}

	return apartments, nil
}

// GetByID returns an apartment by ID.
func (s *ApartmentService) GetByID(ctx context.Context, id int) (*domain.Apartment, error) {
	apartment, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get apartment by id %d: %w", id, ErrNotFound)
		}

		return nil, fmt.Errorf("get apartment by id %d: %w", id, err)
	}

	return apartment, nil
}

// GetByBuildingID returns apartments for a building.
func (s *ApartmentService) GetByBuildingID(ctx context.Context, buildingID int) ([]domain.Apartment, error) {
	apartments, err := s.repo.FindByBuildingID(ctx, buildingID)
	if err != nil {
		return nil, fmt.Errorf("get apartments by building id %d: %w", buildingID, err)
	}

	return apartments, nil
}

// Upsert validates and creates or updates an apartment.
func (s *ApartmentService) Upsert(ctx context.Context, a domain.Apartment) (*domain.Apartment, error) {
	if a.BuildingID == 0 {
		return nil, ErrMissingBuildingID
	}

	if a.Number == "" {
		return nil, ErrMissingApartmentNumber
	}

	if a.Floor == 0 {
		return nil, ErrInvalidFloor
	}

	if a.SqMeters <= 0 {
		return nil, ErrInvalidSqMeters
	}

	apartment, err := s.repo.Upsert(ctx, a)
	if err != nil {
		return nil, fmt.Errorf("upsert apartment: %w", err)
	}

	return apartment, nil
}

// Delete removes an apartment by ID.
func (s *ApartmentService) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("delete apartment %d: %w", id, ErrNotFound)
		}

		return fmt.Errorf("delete apartment %d: %w", id, err)
	}

	return nil
}
