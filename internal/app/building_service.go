// Package app contains the application services (use cases). It depends only
// on the domain and on the repository ports.
package app

import (
	"context"
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

// List returns a page of buildings, optionally with apartments.
func (s *BuildingService) List(ctx context.Context, p port.ListParams, withApartments bool) ([]domain.Building, error) {
	buildings, err := s.repo.FindAll(ctx, p, withApartments)
	if err != nil {
		return nil, fmt.Errorf("list buildings: %w", err)
	}

	return buildings, nil
}

// GetByID returns a building with its apartments.
func (s *BuildingService) GetByID(ctx context.Context, id int) (*domain.Building, error) {
	building, err := s.repo.FindByID(ctx, id, true)
	if err != nil {
		return nil, fmt.Errorf("get building %d: %w", id, err)
	}

	return building, nil
}

// Upsert validates and creates or updates a building. created reports whether
// a new building was inserted.
func (s *BuildingService) Upsert(ctx context.Context, b domain.Building) (*domain.Building, bool, error) {
	if err := b.Validate(); err != nil {
		return nil, false, err
	}

	building, created, err := s.repo.Upsert(ctx, b)
	if err != nil {
		return nil, false, fmt.Errorf("upsert building: %w", err)
	}

	return building, created, nil
}

// Delete removes a building (and, by cascade, its apartments).
func (s *BuildingService) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete building %d: %w", id, err)
	}

	return nil
}
