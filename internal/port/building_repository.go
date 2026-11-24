// Package port declares the interfaces the application layer depends on.
//
// Error contract for every repository method: a missing row is reported as
// domain.ErrNotFound, a uniqueness violation as domain.ErrConflict and a
// broken foreign key as domain.ErrInvalidReference (all may be wrapped).
// Implementations must not leak driver-specific errors such as sql.ErrNoRows.
package port

import (
	"context"

	"github.com/ozaitsev92/bms/internal/domain"
)

// ListParams controls keyset pagination of list queries.
// Results are ordered by ID ascending and start after AfterID.
type ListParams struct {
	Limit   int
	AfterID int
}

// BuildingRepository defines persistence operations for buildings.
type BuildingRepository interface {
	FindAll(ctx context.Context, p ListParams, withApartments bool) ([]domain.Building, error)
	FindByID(ctx context.Context, id int, withApartments bool) (*domain.Building, error)
	Upsert(ctx context.Context, b domain.Building) (result *domain.Building, created bool, err error)
	Delete(ctx context.Context, id int) error
}
