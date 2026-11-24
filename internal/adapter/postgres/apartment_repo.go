package postgres

import (
	"context"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
	"github.com/ozaitsev92/bms/models"
)

// ApartmentRepository implements port.ApartmentRepository on PostgreSQL.
type ApartmentRepository struct {
	exec boil.ContextExecutor
}

// NewApartmentRepo creates an apartment repository. exec may be a *sql.DB or a *sql.Tx.
func NewApartmentRepo(exec boil.ContextExecutor) *ApartmentRepository {
	return &ApartmentRepository{exec: exec}
}

// FindAll returns a page of apartments ordered by ID.
func (r *ApartmentRepository) FindAll(ctx context.Context, p port.ListParams) ([]domain.Apartment, error) {
	rows, err := models.Apartments(pageMods(p)...).All(ctx, r.exec)
	if err != nil {
		return nil, fmt.Errorf("find apartments: %w", mapError(err))
	}

	return toApartments(rows), nil
}

// FindByID returns an apartment by ID, or domain.ErrNotFound.
func (r *ApartmentRepository) FindByID(ctx context.Context, id int) (*domain.Apartment, error) {
	row, err := models.FindApartment(ctx, r.exec, id)
	if err != nil {
		return nil, fmt.Errorf("find apartment %d: %w", id, mapError(err))
	}

	result := toApartment(row)

	return &result, nil
}

// FindByBuildingID returns a page of apartments of a building, or
// domain.ErrNotFound if the building does not exist.
func (r *ApartmentRepository) FindByBuildingID(ctx context.Context, buildingID int, p port.ListParams) ([]domain.Apartment, error) {
	mods := append(pageMods(p), models.ApartmentWhere.BuildingID.EQ(buildingID))

	rows, err := models.Apartments(mods...).All(ctx, r.exec)
	if err != nil {
		return nil, fmt.Errorf("find apartments of building %d: %w", buildingID, mapError(err))
	}

	if len(rows) == 0 {
		// Only pay for the existence check when it can change the answer.
		exists, err := models.BuildingExists(ctx, r.exec, buildingID)
		if err != nil {
			return nil, fmt.Errorf("check building %d exists: %w", buildingID, mapError(err))
		}
		if !exists {
			return nil, fmt.Errorf("find apartments of building %d: %w", buildingID, domain.ErrNotFound)
		}
	}

	return toApartments(rows), nil
}

const (
	insertOrUpdateApartmentByNumber = `
INSERT INTO apartment (building_id, number, floor, sq_meters) VALUES ($1, $2, $3, $4)
ON CONFLICT (building_id, number) DO UPDATE
SET floor = EXCLUDED.floor, sq_meters = EXCLUDED.sq_meters, updated_at = now()
RETURNING id, building_id, number, floor, sq_meters, created_at, updated_at, (xmax = 0) AS inserted`

	updateApartmentByID = `
UPDATE apartment SET building_id = $2, number = $3, floor = $4, sq_meters = $5, updated_at = now()
WHERE id = $1
RETURNING id, building_id, number, floor, sq_meters, created_at, updated_at`
)

// Upsert has the same semantics as BuildingRepository.Upsert, keyed by (building_id, number).
func (r *ApartmentRepository) Upsert(ctx context.Context, a domain.Apartment) (*domain.Apartment, bool, error) {
	var out domain.Apartment

	if a.ID != 0 {
		err := r.exec.QueryRowContext(ctx, updateApartmentByID, a.ID, a.BuildingID, a.Number, a.Floor, a.SqMeters).
			Scan(&out.ID, &out.BuildingID, &out.Number, &out.Floor, &out.SqMeters, &out.CreatedAt, &out.UpdatedAt)
		if err != nil {
			return nil, false, fmt.Errorf("update apartment %d: %w", a.ID, mapError(err))
		}

		return &out, false, nil
	}

	var inserted bool
	err := r.exec.QueryRowContext(ctx, insertOrUpdateApartmentByNumber, a.BuildingID, a.Number, a.Floor, a.SqMeters).
		Scan(&out.ID, &out.BuildingID, &out.Number, &out.Floor, &out.SqMeters, &out.CreatedAt, &out.UpdatedAt, &inserted)
	if err != nil {
		return nil, false, fmt.Errorf("upsert apartment by number: %w", mapError(err))
	}

	return &out, inserted, nil
}

// Delete removes an apartment in a single statement, or returns domain.ErrNotFound.
func (r *ApartmentRepository) Delete(ctx context.Context, id int) error {
	n, err := models.Apartments(models.ApartmentWhere.ID.EQ(id)).DeleteAll(ctx, r.exec)
	if err != nil {
		return fmt.Errorf("delete apartment %d: %w", id, mapError(err))
	}
	if n == 0 {
		return fmt.Errorf("delete apartment %d: %w", id, domain.ErrNotFound)
	}

	return nil
}

func toApartments(rows models.ApartmentSlice) []domain.Apartment {
	out := make([]domain.Apartment, len(rows))
	for i, row := range rows {
		out[i] = toApartment(row)
	}

	return out
}

func toApartment(m *models.Apartment) domain.Apartment {
	return domain.Apartment{
		ID:         m.ID,
		BuildingID: m.BuildingID,
		Number:     m.Number,
		Floor:      m.Floor,
		SqMeters:   m.SQMeters,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
