package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

// BuildingService coordinates building-related business operations.
type BuildingService struct {
	repo port.BuildingRepository
}

// NewBuildingService creates a building application service.
func NewBuildingService(repo port.BuildingRepository) *BuildingService {
	return &BuildingService{repo: repo}
}

// GetAll returns all buildings, optionally with apartments.
func (s *BuildingService) GetAll(ctx context.Context, withApartments bool) ([]domain.Building, error) {
	buildings, err := s.repo.FindAll(ctx, withApartments)
	if err != nil {
		return nil, fmt.Errorf("get all buildings: %w", err)
	}

	return buildings, nil
}

// GetByID returns a building by ID.
func (s *BuildingService) GetByID(ctx context.Context, id int) (*domain.Building, error) {
	building, err := s.repo.FindByID(ctx, id, true)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get building by id %d: %w", id, ErrNotFound)
		}

		return nil, fmt.Errorf("get building by id %d: %w", id, err)
	}

	return building, nil
}

// Upsert validates and creates or updates a building.
func (s *BuildingService) Upsert(ctx context.Context, b domain.Building) (*domain.Building, error) {
	if b.Name == "" {
		return nil, ErrMissingBuildingName
	}

	if b.Address == "" {
		return nil, ErrMissingBuildingAddress
	}

	building, err := s.repo.Upsert(ctx, b)
	if err != nil {
		return nil, fmt.Errorf("upsert building: %w", err)
	}

	return building, nil
}

// Delete removes a building by ID.
func (s *BuildingService) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("delete building %d: %w", id, ErrNotFound)
		}

		return fmt.Errorf("delete building %d: %w", id, err)
	}

	return nil
}
