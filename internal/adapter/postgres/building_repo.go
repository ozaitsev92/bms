package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/models"
)

// BuildingRepository implements building persistence in PostgreSQL.
type BuildingRepository struct {
	db *sql.DB
}

// NewBuildingRepo creates a PostgreSQL building repository.
func NewBuildingRepo(db *sql.DB) *BuildingRepository {
	return &BuildingRepository{db: db}
}

// FindAll returns all buildings, optionally with apartments loaded.
func (r *BuildingRepository) FindAll(ctx context.Context, withApartments bool) ([]domain.Building, error) {
	mods := []qm.QueryMod{}
	if withApartments {
		mods = append(mods, qm.Load(models.BuildingRels.Apartments))
	}

	rows, err := models.Buildings(mods...).All(ctx, r.db)
	if err != nil {
		return nil, fmt.Errorf("find all buildings: %w", err)
	}

	out := make([]domain.Building, len(rows))
	for i, row := range rows {
		out[i] = toBuilding(row)
	}

	return out, nil
}

// FindByID returns a building by ID.
func (r *BuildingRepository) FindByID(ctx context.Context, id int, withApartments bool) (*domain.Building, error) {
	mods := []qm.QueryMod{qm.Where("id = ?", id)}
	if withApartments {
		mods = append(mods, qm.Load(models.BuildingRels.Apartments))
	}

	row, err := models.Buildings(mods...).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("find building by id %d: %w", id, sql.ErrNoRows)
		}

		return nil, fmt.Errorf("find building by id %d: %w", id, err)
	}

	result := toBuilding(row)

	return &result, nil
}

// Upsert creates or updates a building.
func (r *BuildingRepository) Upsert(ctx context.Context, b domain.Building) (*domain.Building, error) {
	row := &models.Building{
		Name:    b.Name,
		Address: b.Address,
	}

	conflictCols := []string{"name"}
	if b.ID != 0 {
		row.ID = b.ID
		conflictCols = []string{"id"}
	}

	err := row.Upsert(
		ctx,
		r.db,
		true,
		conflictCols,
		boil.Infer(),
		boil.Infer(),
	)
	if err != nil {
		return nil, fmt.Errorf("upsert building: %w", err)
	}

	result := toBuilding(row)

	return &result, nil
}

// Delete removes a building by ID.
func (r *BuildingRepository) Delete(ctx context.Context, id int) error {
	row, err := models.FindBuilding(ctx, r.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("delete building %d: %w", id, sql.ErrNoRows)
		}

		return fmt.Errorf("delete building %d: %w", id, err)
	}

	if _, err := row.Delete(ctx, r.db); err != nil {
		return fmt.Errorf("delete building %d: %w", id, err)
	}

	return nil
}

func toBuilding(m *models.Building) domain.Building {
	b := domain.Building{
		ID:      m.ID,
		Name:    m.Name,
		Address: m.Address,
	}
	if m.R != nil {
		for _, a := range m.R.Apartments {
			b.Apartments = append(b.Apartments, toApartment(a))
		}
	}

	return b
}
