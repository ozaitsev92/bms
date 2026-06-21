package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/adapter/postgres"
	"github.com/ozaitsev92/bms/internal/domain"
)

func TestApartmentRepository_CRUD(t *testing.T) {
	ctx := context.Background()
	_, db := setupPostgresDB(t)
	buildingRepo := postgres.NewBuildingRepo(db)
	repo := postgres.NewApartmentRepo(db)

	b, err := buildingRepo.Upsert(ctx, domain.Building{
		Name:    "FK Building",
		Address: "1 FK Street",
	})
	require.NoError(t, err)

	a, err := repo.Upsert(ctx, domain.Apartment{
		BuildingID: b.ID,
		Number:     "101",
		Floor:      1,
		SqMeters:   55,
	})
	require.NoError(t, err)
	assert.NotZero(t, a.ID)
	assert.Equal(t, b.ID, a.BuildingID)
	assert.Equal(t, "101", a.Number)
	assert.Equal(t, 1, a.Floor)
	assert.Equal(t, 55, a.SqMeters)

	found, err := repo.FindByID(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, a.ID, found.ID)
	assert.Equal(t, a.Number, found.Number)

	updated, err := repo.Upsert(ctx, domain.Apartment{
		ID:         a.ID,
		BuildingID: b.ID,
		Number:     "101A",
		Floor:      2,
		SqMeters:   70,
	})
	require.NoError(t, err)
	assert.Equal(t, a.ID, updated.ID)
	assert.Equal(t, "101A", updated.Number)
	assert.Equal(t, 2, updated.Floor)
	assert.Equal(t, 70, updated.SqMeters)

	err = repo.Delete(ctx, a.ID)
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, a.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestApartmentRepository_FindAll(t *testing.T) {
	ctx := context.Background()
	_, db := setupPostgresDB(t)
	buildingRepo := postgres.NewBuildingRepo(db)
	repo := postgres.NewApartmentRepo(db)

	b, err := buildingRepo.Upsert(ctx, domain.Building{
		Name: "Apt Building", Address: "2 Apt Ave",
	})
	require.NoError(t, err)

	for _, a := range []domain.Apartment{
		{BuildingID: b.ID, Number: "101", Floor: 1, SqMeters: 50},
		{BuildingID: b.ID, Number: "102", Floor: 1, SqMeters: 55},
	} {
		_, err := repo.Upsert(ctx, a)
		require.NoError(t, err)
	}

	apartments, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, apartments, 2)
}

func TestApartmentRepository_FindByBuildingID(t *testing.T) {
	ctx := context.Background()
	_, db := setupPostgresDB(t)
	buildingRepo := postgres.NewBuildingRepo(db)
	repo := postgres.NewApartmentRepo(db)

	b1, _ := buildingRepo.Upsert(ctx, domain.Building{Name: "B1", Address: "1 St"})
	b2, _ := buildingRepo.Upsert(ctx, domain.Building{Name: "B2", Address: "2 St"})

	for _, a := range []domain.Apartment{
		{BuildingID: b1.ID, Number: "101", Floor: 1, SqMeters: 50},
		{BuildingID: b1.ID, Number: "102", Floor: 1, SqMeters: 55},
		{BuildingID: b2.ID, Number: "201", Floor: 2, SqMeters: 70},
	} {
		_, err := repo.Upsert(ctx, a)
		require.NoError(t, err)
	}

	apts, err := repo.FindByBuildingID(ctx, b1.ID)
	require.NoError(t, err)
	assert.Len(t, apts, 2)

	for _, a := range apts {
		assert.Equal(t, b1.ID, a.BuildingID)
	}
}

func TestApartmentRepository_FindByID_NotFound(t *testing.T) {
	ctx := context.Background()
	_, db := setupPostgresDB(t)
	apartmentRepo := postgres.NewApartmentRepo(db)

	_, err := apartmentRepo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestApartmentRepository_UniqueConstraint(t *testing.T) {
	ctx := context.Background()
	_, db := setupPostgresDB(t)
	buildingRepo := postgres.NewBuildingRepo(db)
	repo := postgres.NewApartmentRepo(db)

	b, err := buildingRepo.Upsert(ctx, domain.Building{Name: "Unique B", Address: "3 St"})
	require.NoError(t, err)

	a1, err := repo.Upsert(ctx, domain.Apartment{
		BuildingID: b.ID, Number: "101", Floor: 1, SqMeters: 50,
	})
	require.NoError(t, err)

	a2, err := repo.Upsert(ctx, domain.Apartment{
		BuildingID: b.ID, Number: "101", Floor: 2, SqMeters: 60,
	})
	require.NoError(t, err)

	assert.Equal(t, a1.ID, a2.ID)
	assert.Equal(t, 2, a2.Floor)
	assert.Equal(t, 60, a2.SqMeters)

	all, err := repo.FindByBuildingID(ctx, b.ID)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}
