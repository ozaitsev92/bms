package postgres

import (
	"context"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
	"github.com/ozaitsev92/bms/models"
)

// BuildingRepository implements port.BuildingRepository on PostgreSQL.
type BuildingRepository struct {
	exec boil.ContextExecutor
}

// NewBuildingRepo creates a building repository. exec may be a *sql.DB or a
// *sql.Tx, which lets callers (and tests) run it inside a transaction.
func NewBuildingRepo(exec boil.ContextExecutor) *BuildingRepository {
	return &BuildingRepository{exec: exec}
}

// FindAll returns a page of buildings ordered by ID, optionally with apartments.
func (r *BuildingRepository) FindAll(ctx context.Context, p port.ListParams, withApartments bool) ([]domain.Building, error) {
	mods := pageMods(p)
	if withApartments {
		mods = append(mods, qm.Load(models.BuildingRels.Apartments, qm.OrderBy("id ASC")))
	}

	rows, err := models.Buildings(mods...).All(ctx, r.exec)
	if err != nil {
		return nil, fmt.Errorf("find buildings: %w", mapError(err))
	}

	out := make([]domain.Building, len(rows))
	for i, row := range rows {
		out[i] = toBuilding(row, withApartments)
	}

	return out, nil
}

// FindByID returns a building by ID, or domain.ErrNotFound.
func (r *BuildingRepository) FindByID(ctx context.Context, id int, withApartments bool) (*domain.Building, error) {
	mods := []qm.QueryMod{models.BuildingWhere.ID.EQ(id)}
	if withApartments {
		mods = append(mods, qm.Load(models.BuildingRels.Apartments, qm.OrderBy("id ASC")))
	}

	row, err := models.Buildings(mods...).One(ctx, r.exec)
	if err != nil {
		return nil, fmt.Errorf("find building %d: %w", id, mapError(err))
	}

	result := toBuilding(row, withApartments)

	return &result, nil
}

const (
	insertOrUpdateBuildingByName = `
INSERT INTO building (name, address) VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET address = EXCLUDED.address, updated_at = now()
RETURNING id, name, address, created_at, updated_at, (xmax = 0) AS inserted`

	updateBuildingByID = `
UPDATE building SET name = $2, address = $3, updated_at = now()
WHERE id = $1
RETURNING id, name, address, created_at, updated_at`
)

// Upsert updates the building by ID when b.ID is set and never inserts in that
// case, so clients cannot pick primary keys or desynchronise the sequence.
// Without an ID it inserts, or updates the row with the same unique name.
func (r *BuildingRepository) Upsert(ctx context.Context, b domain.Building) (*domain.Building, bool, error) {
	var out domain.Building

	if b.ID != 0 {
		err := r.exec.QueryRowContext(ctx, updateBuildingByID, b.ID, b.Name, b.Address).
			Scan(&out.ID, &out.Name, &out.Address, &out.CreatedAt, &out.UpdatedAt)
		if err != nil {
			return nil, false, fmt.Errorf("update building %d: %w", b.ID, mapError(err))
		}

		return &out, false, nil
	}

	var inserted bool
	err := r.exec.QueryRowContext(ctx, insertOrUpdateBuildingByName, b.Name, b.Address).
		Scan(&out.ID, &out.Name, &out.Address, &out.CreatedAt, &out.UpdatedAt, &inserted)
	if err != nil {
		return nil, false, fmt.Errorf("upsert building by name: %w", mapError(err))
	}

	return &out, inserted, nil
}

// Delete removes a building in a single statement, or returns domain.ErrNotFound.
func (r *BuildingRepository) Delete(ctx context.Context, id int) error {
	n, err := models.Buildings(models.BuildingWhere.ID.EQ(id)).DeleteAll(ctx, r.exec)
	if err != nil {
		return fmt.Errorf("delete building %d: %w", id, mapError(err))
	}
	if n == 0 {
		return fmt.Errorf("delete building %d: %w", id, domain.ErrNotFound)
	}

	return nil
}

// toBuilding maps a model to the domain. When apartments were loaded the
// slice is non-nil (possibly empty) so callers can tell "none" from "not loaded".
func toBuilding(m *models.Building, withApartments bool) domain.Building {
	b := domain.Building{
		ID:        m.ID,
		Name:      m.Name,
		Address:   m.Address,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if withApartments {
		b.Apartments = []domain.Apartment{}
		if m.R != nil {
			b.Apartments = make([]domain.Apartment, len(m.R.Apartments))
			for i, a := range m.R.Apartments {
				b.Apartments[i] = toApartment(a)
			}
		}
	}

	return b
}
