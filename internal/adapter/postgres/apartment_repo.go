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

// ApartmentRepository implements apartment persistence in PostgreSQL.
type ApartmentRepository struct {
	db *sql.DB
}

// NewApartmentRepo creates a PostgreSQL apartment repository.
func NewApartmentRepo(db *sql.DB) *ApartmentRepository {
	return &ApartmentRepository{db: db}
}

// FindAll returns all apartments.
func (r *ApartmentRepository) FindAll(ctx context.Context) ([]domain.Apartment, error) {
	rows, err := models.Apartments().All(ctx, r.db)
	if err != nil {
		return nil, fmt.Errorf("find all apartments: %w", err)
	}

	out := make([]domain.Apartment, len(rows))
	for i, row := range rows {
		out[i] = toApartment(row)
	}

	return out, nil
}

// FindByID returns an apartment by ID.
func (r *ApartmentRepository) FindByID(ctx context.Context, id int) (*domain.Apartment, error) {
	row, err := models.FindApartment(ctx, r.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("find apartment by id %d: %w", id, sql.ErrNoRows)
		}
		return nil, fmt.Errorf("find apartment by id %d: %w", id, err)
	}

	result := toApartment(row)

	return &result, nil
}

// FindByBuildingID returns apartments for a building ID.
func (r *ApartmentRepository) FindByBuildingID(ctx context.Context, buildingID int) ([]domain.Apartment, error) {
	rows, err := models.Apartments(
		qm.Where("building_id = ?", buildingID),
	).All(ctx, r.db)
	if err != nil {
		return nil, fmt.Errorf("find apartments by building id %d: %w", buildingID, err)
	}

	out := make([]domain.Apartment, len(rows))
	for i, row := range rows {
		out[i] = toApartment(row)
	}

	return out, nil
}

// Upsert creates or updates an apartment.
func (r *ApartmentRepository) Upsert(ctx context.Context, a domain.Apartment) (*domain.Apartment, error) {
	row := &models.Apartment{
		BuildingID: a.BuildingID,
		Number:     a.Number,
		Floor:      a.Floor,
		SQMeters:   a.SqMeters,
	}

	conflictCols := []string{"building_id", "number"}
	if a.ID != 0 {
		row.ID = a.ID
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
		return nil, fmt.Errorf("upsert apartment: %w", err)
	}

	result := toApartment(row)

	return &result, nil
}

// Delete removes an apartment by ID.
func (r *ApartmentRepository) Delete(ctx context.Context, id int) error {
	row, err := models.FindApartment(ctx, r.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("delete apartment %d: %w", id, sql.ErrNoRows)
		}

		return fmt.Errorf("delete apartment %d: %w", id, err)
	}

	if _, err := row.Delete(ctx, r.db); err != nil {
		return fmt.Errorf("delete apartment %d: %w", id, err)
	}

	return nil
}

func toApartment(m *models.Apartment) domain.Apartment {
	return domain.Apartment{
		ID:         m.ID,
		BuildingID: m.BuildingID,
		Number:     m.Number,
		Floor:      m.Floor,
		SqMeters:   m.SQMeters,
	}
}
