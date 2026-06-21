package port

import (
	"context"

	"github.com/ozaitsev92/bms/internal/domain"
)

// BuildingRepository defines persistence operations for buildings.
type BuildingRepository interface {
	FindAll(ctx context.Context, withApartments bool) ([]domain.Building, error)
	FindByID(ctx context.Context, id int, withApartments bool) (*domain.Building, error)
	Upsert(ctx context.Context, b domain.Building) (*domain.Building, error)
	Delete(ctx context.Context, id int) error
}
