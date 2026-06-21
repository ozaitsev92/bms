package port

import (
	"context"

	"github.com/ozaitsev92/bms/internal/domain"
)

// ApartmentRepository defines persistence operations for apartments.
type ApartmentRepository interface {
	FindAll(ctx context.Context) ([]domain.Apartment, error)
	FindByID(ctx context.Context, id int) (*domain.Apartment, error)
	FindByBuildingID(ctx context.Context, buildingID int) ([]domain.Apartment, error)
	Upsert(ctx context.Context, a domain.Apartment) (*domain.Apartment, error)
	Delete(ctx context.Context, id int) error
}
