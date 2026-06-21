package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/adapter/postgres"
	"github.com/ozaitsev92/bms/internal/domain"
)

func TestBuildingRepository_CRUD(t *testing.T) {
	ctx := context.Background()
	repo, _ := setupPostgresDB(t)

	b, err := repo.Upsert(ctx, domain.Building{
		Name:    "Tower A",
		Address: "123 Main St",
	})
	require.NoError(t, err)
	assert.NotZero(t, b.ID)
	assert.Equal(t, "Tower A", b.Name)
	assert.Equal(t, "123 Main St", b.Address)

	found, err := repo.FindByID(ctx, b.ID, false)
	require.NoError(t, err)
	assert.Equal(t, b.ID, found.ID)
	assert.Equal(t, b.Name, found.Name)
	assert.Equal(t, b.Address, found.Address)

	updated, err := repo.Upsert(ctx, domain.Building{
		ID:      b.ID,
		Name:    "Tower A Updated",
		Address: "456 New Ave",
	})
	require.NoError(t, err)
	assert.Equal(t, b.ID, updated.ID)
	assert.Equal(t, "Tower A Updated", updated.Name)

	err = repo.Delete(ctx, b.ID)
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, b.ID, false)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestBuildingRepository_FindAll(t *testing.T) {
	ctx := context.Background()
	repo, _ := setupPostgresDB(t)

	for _, b := range []domain.Building{
		{Name: "Tower A", Address: "123 Main St"},
		{Name: "Tower B", Address: "456 Side Ave"},
	} {
		_, err := repo.Upsert(ctx, b)
		require.NoError(t, err)
	}

	buildings, err := repo.FindAll(ctx, false)
	require.NoError(t, err)
	assert.Len(t, buildings, 2)
}

func TestBuildingRepository_FindAll_WithApartments(t *testing.T) {
	ctx := context.Background()
	buildingRepo, db := setupPostgresDB(t)
	apartmentRepo := postgres.NewApartmentRepo(db)

	b, err := buildingRepo.Upsert(ctx, domain.Building{
		Name:    "Tower C",
		Address: "789 Cross Rd",
	})
	require.NoError(t, err)

	_, err = apartmentRepo.Upsert(ctx, domain.Apartment{
		BuildingID: b.ID,
		Number:     "101",
		Floor:      1,
		SqMeters:   50,
	})
	require.NoError(t, err)

	buildings, err := buildingRepo.FindAll(ctx, true)
	require.NoError(t, err)
	require.Len(t, buildings, 1)
	assert.Len(t, buildings[0].Apartments, 1)
	assert.Equal(t, "101", buildings[0].Apartments[0].Number)
}

func TestBuildingRepository_FindByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo, _ := setupPostgresDB(t)

	_, err := repo.FindByID(ctx, 99999, false)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestBuildingRepository_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	repo, _ := setupPostgresDB(t)

	err := repo.Delete(ctx, 99999)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestBuildingRepository_UniqueNameUpsert(t *testing.T) {
	ctx := context.Background()
	repo, _ := setupPostgresDB(t)

	b1, err := repo.Upsert(ctx, domain.Building{
		Name:    "Tower A",
		Address: "123 Main St",
	})
	require.NoError(t, err)

	b2, err := repo.Upsert(ctx, domain.Building{
		Name:    "Tower A",
		Address: "456 Updated Ave",
	})
	require.NoError(t, err)

	assert.Equal(t, b1.ID, b2.ID)
	assert.Equal(t, "456 Updated Ave", b2.Address)

	all, err := repo.FindAll(ctx, false)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, b1.ID, all[0].ID)
	assert.Equal(t, "Tower A", all[0].Name)
	assert.Equal(t, "456 Updated Ave", all[0].Address)
}
