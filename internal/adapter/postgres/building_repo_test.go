package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/adapter/postgres"
	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

func TestBuildingRepository_Upsert_InsertThenUpdateByName(t *testing.T) {
	repo := postgres.NewBuildingRepo(freshDB(t))

	b1, created, err := repo.Upsert(t.Context(), domain.Building{Name: "Tower A", Address: "1 St"})
	require.NoError(t, err)
	assert.True(t, created)
	assert.NotZero(t, b1.ID)
	assert.False(t, b1.CreatedAt.IsZero())

	b2, created, err := repo.Upsert(t.Context(), domain.Building{Name: "Tower A", Address: "2 St"})
	require.NoError(t, err)
	assert.False(t, created, "existing name must be reported as an update")
	assert.Equal(t, b1.ID, b2.ID)
	assert.Equal(t, "2 St", b2.Address)
	assert.Equal(t, b1.CreatedAt, b2.CreatedAt)
	assert.False(t, b2.UpdatedAt.Before(b1.UpdatedAt))
}

func TestBuildingRepository_Upsert_UpdateByID(t *testing.T) {
	repo := postgres.NewBuildingRepo(freshDB(t))
	b := mustCreateBuilding(t, repo, "Old")

	updated, created, err := repo.Upsert(t.Context(), domain.Building{ID: b.ID, Name: "New", Address: "9 St"})
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, b.ID, updated.ID)
	assert.Equal(t, "New", updated.Name)

	found, err := repo.FindByID(t.Context(), b.ID, false)
	require.NoError(t, err)
	assert.Equal(t, "New", found.Name)
	assert.Equal(t, "9 St", found.Address)
}

// Regression test: an unknown explicit ID used to be inserted verbatim, which
// left the SERIAL sequence behind and made later creates fail.
func TestBuildingRepository_Upsert_UnknownID_ReturnsNotFoundAndKeepsSequence(t *testing.T) {
	repo := postgres.NewBuildingRepo(freshDB(t))

	_, _, err := repo.Upsert(t.Context(), domain.Building{ID: 3, Name: "Ghost", Address: "x"})
	require.ErrorIs(t, err, domain.ErrNotFound)

	for i := 1; i <= 5; i++ {
		b := mustCreateBuilding(t, repo, fmt.Sprintf("B%d", i))
		assert.Equal(t, i, b.ID)
	}
}

func TestBuildingRepository_Upsert_RenameToTakenName_ReturnsConflict(t *testing.T) {
	repo := postgres.NewBuildingRepo(freshDB(t))
	mustCreateBuilding(t, repo, "Taken")
	other := mustCreateBuilding(t, repo, "Other")

	_, _, err := repo.Upsert(t.Context(), domain.Building{ID: other.ID, Name: "Taken", Address: "x"})
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestBuildingRepository_FindByID_NotFound(t *testing.T) {
	repo := postgres.NewBuildingRepo(freshDB(t))

	_, err := repo.FindByID(t.Context(), 999, true)
	require.ErrorIs(t, err, domain.ErrNotFound)
	assert.NotErrorIs(t, err, sql.ErrNoRows, "driver errors must not leak through the port")
}

func TestBuildingRepository_FindByID_WithApartments(t *testing.T) {
	db := freshDB(t)
	repo := postgres.NewBuildingRepo(db)
	aptRepo := postgres.NewApartmentRepo(db)
	empty := mustCreateBuilding(t, repo, "Empty")
	full := mustCreateBuilding(t, repo, "Full")
	mustCreateApartment(t, aptRepo, full.ID, "2")
	mustCreateApartment(t, aptRepo, full.ID, "1")

	got, err := repo.FindByID(t.Context(), empty.ID, true)
	require.NoError(t, err)
	assert.NotNil(t, got.Apartments, "loaded-but-empty must be an empty slice, not nil")
	assert.Empty(t, got.Apartments)

	got, err = repo.FindByID(t.Context(), full.ID, true)
	require.NoError(t, err)
	require.Len(t, got.Apartments, 2)
	assert.Less(t, got.Apartments[0].ID, got.Apartments[1].ID, "apartments ordered by id")

	got, err = repo.FindByID(t.Context(), full.ID, false)
	require.NoError(t, err)
	assert.Nil(t, got.Apartments)
}

func TestBuildingRepository_FindAll_PaginationOrdered(t *testing.T) {
	repo := postgres.NewBuildingRepo(freshDB(t))
	for i := 1; i <= 5; i++ {
		mustCreateBuilding(t, repo, fmt.Sprintf("B%d", i))
	}

	ids := func(bs []domain.Building) []int {
		out := make([]int, len(bs))
		for i, b := range bs {
			out[i] = b.ID
		}
		return out
	}

	page, err := repo.FindAll(t.Context(), port.ListParams{Limit: 2}, false)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, ids(page))

	page, err = repo.FindAll(t.Context(), port.ListParams{Limit: 2, AfterID: 2}, false)
	require.NoError(t, err)
	assert.Equal(t, []int{3, 4}, ids(page))

	page, err = repo.FindAll(t.Context(), port.ListParams{Limit: 2, AfterID: 4}, false)
	require.NoError(t, err)
	assert.Equal(t, []int{5}, ids(page))

	page, err = repo.FindAll(t.Context(), port.ListParams{}, false)
	require.NoError(t, err)
	assert.Len(t, page, 5, "zero limit means no limit at the repository level")
}

func TestBuildingRepository_FindAll_WithApartments(t *testing.T) {
	db := freshDB(t)
	repo := postgres.NewBuildingRepo(db)
	aptRepo := postgres.NewApartmentRepo(db)
	b := mustCreateBuilding(t, repo, "With")
	mustCreateBuilding(t, repo, "Without")
	mustCreateApartment(t, aptRepo, b.ID, "101")

	all, err := repo.FindAll(t.Context(), port.ListParams{}, true)
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Len(t, all[0].Apartments, 1)
	assert.NotNil(t, all[1].Apartments)
	assert.Empty(t, all[1].Apartments)
}

func TestBuildingRepository_Delete(t *testing.T) {
	repo := postgres.NewBuildingRepo(freshDB(t))
	b := mustCreateBuilding(t, repo, "Doomed")

	require.NoError(t, repo.Delete(t.Context(), b.ID))

	_, err := repo.FindByID(t.Context(), b.ID, false)
	require.ErrorIs(t, err, domain.ErrNotFound)

	require.ErrorIs(t, repo.Delete(t.Context(), b.ID), domain.ErrNotFound, "second delete")
}

func TestBuildingRepository_Delete_CascadesApartments(t *testing.T) {
	db := freshDB(t)
	repo := postgres.NewBuildingRepo(db)
	aptRepo := postgres.NewApartmentRepo(db)
	b := mustCreateBuilding(t, repo, "Cascade")
	a := mustCreateApartment(t, aptRepo, b.ID, "1")

	require.NoError(t, repo.Delete(t.Context(), b.ID))

	_, err := aptRepo.FindByID(t.Context(), a.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestBuildingRepository_CanceledContext_ReturnsError(t *testing.T) {
	repo := postgres.NewBuildingRepo(freshDB(t))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.FindAll(ctx, port.ListParams{}, false)
	require.ErrorIs(t, err, context.Canceled)
}

func TestBuildingRepository_WorksInsideTransaction(t *testing.T) {
	db := freshDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)

	mustCreateBuilding(t, postgres.NewBuildingRepo(tx), "Rolled back")
	require.NoError(t, tx.Rollback())

	all, err := postgres.NewBuildingRepo(db).FindAll(t.Context(), port.ListParams{}, false)
	require.NoError(t, err)
	assert.Empty(t, all)
}
