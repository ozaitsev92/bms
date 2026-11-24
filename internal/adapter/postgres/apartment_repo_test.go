package postgres_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/adapter/postgres"
	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

func newRepos(t *testing.T) (*postgres.BuildingRepository, *postgres.ApartmentRepository) {
	t.Helper()
	db := freshDB(t)

	return postgres.NewBuildingRepo(db), postgres.NewApartmentRepo(db)
}

func TestApartmentRepository_Upsert_InsertThenUpdateByNumber(t *testing.T) {
	buildings, repo := newRepos(t)
	b := mustCreateBuilding(t, buildings, "B")

	a1, created, err := repo.Upsert(t.Context(), domain.Apartment{BuildingID: b.ID, Number: "101", Floor: 1, SqMeters: 50})
	require.NoError(t, err)
	assert.True(t, created)

	a2, created, err := repo.Upsert(t.Context(), domain.Apartment{BuildingID: b.ID, Number: "101", Floor: 2, SqMeters: 60})
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, a1.ID, a2.ID)
	assert.Equal(t, 2, a2.Floor)
	assert.Equal(t, 60, a2.SqMeters)
}

func TestApartmentRepository_Upsert_UpdateByID(t *testing.T) {
	buildings, repo := newRepos(t)
	b1 := mustCreateBuilding(t, buildings, "B1")
	b2 := mustCreateBuilding(t, buildings, "B2")
	a := mustCreateApartment(t, repo, b1.ID, "101")

	updated, created, err := repo.Upsert(t.Context(), domain.Apartment{ID: a.ID, BuildingID: b2.ID, Number: "101A", Floor: 0, SqMeters: 70})
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, a.ID, updated.ID)
	assert.Equal(t, b2.ID, updated.BuildingID)
	assert.Equal(t, "101A", updated.Number)
	assert.Equal(t, 0, updated.Floor, "ground floor is valid")
}

func TestApartmentRepository_Upsert_UnknownID_ReturnsNotFound(t *testing.T) {
	buildings, repo := newRepos(t)
	b := mustCreateBuilding(t, buildings, "B")

	_, _, err := repo.Upsert(t.Context(), domain.Apartment{ID: 42, BuildingID: b.ID, Number: "1", Floor: 1, SqMeters: 10})
	require.ErrorIs(t, err, domain.ErrNotFound)

	a := mustCreateApartment(t, repo, b.ID, "1")
	assert.Equal(t, 1, a.ID, "sequence untouched")
}

func TestApartmentRepository_Upsert_UnknownBuilding_ReturnsInvalidReference(t *testing.T) {
	buildings, repo := newRepos(t)

	_, _, err := repo.Upsert(t.Context(), domain.Apartment{BuildingID: 999, Number: "1", Floor: 1, SqMeters: 10})
	require.ErrorIs(t, err, domain.ErrInvalidReference, "insert path")

	b := mustCreateBuilding(t, buildings, "B")
	a := mustCreateApartment(t, repo, b.ID, "1")
	_, _, err = repo.Upsert(t.Context(), domain.Apartment{ID: a.ID, BuildingID: 999, Number: "1", Floor: 1, SqMeters: 10})
	require.ErrorIs(t, err, domain.ErrInvalidReference, "update path")
}

func TestApartmentRepository_Upsert_NumberTakenViaUpdate_ReturnsConflict(t *testing.T) {
	buildings, repo := newRepos(t)
	b := mustCreateBuilding(t, buildings, "B")
	mustCreateApartment(t, repo, b.ID, "1")
	a2 := mustCreateApartment(t, repo, b.ID, "2")

	_, _, err := repo.Upsert(t.Context(), domain.Apartment{ID: a2.ID, BuildingID: b.ID, Number: "1", Floor: 1, SqMeters: 10})
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestApartmentRepository_CheckConstraint_SqMeters(t *testing.T) {
	buildings, repo := newRepos(t)
	b := mustCreateBuilding(t, buildings, "B")

	// The domain rejects this first; the database constraint is the backstop.
	_, _, err := repo.Upsert(t.Context(), domain.Apartment{BuildingID: b.ID, Number: "1", Floor: 1, SqMeters: 0})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestApartmentRepository_FindByID(t *testing.T) {
	buildings, repo := newRepos(t)
	b := mustCreateBuilding(t, buildings, "B")
	a := mustCreateApartment(t, repo, b.ID, "7")

	got, err := repo.FindByID(t.Context(), a.ID)
	require.NoError(t, err)
	assert.Equal(t, *a, *got)

	_, err = repo.FindByID(t.Context(), 999)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestApartmentRepository_FindAll_PaginationOrdered(t *testing.T) {
	buildings, repo := newRepos(t)
	b := mustCreateBuilding(t, buildings, "B")
	for i := 1; i <= 3; i++ {
		mustCreateApartment(t, repo, b.ID, fmt.Sprint(i))
	}

	page, err := repo.FindAll(t.Context(), port.ListParams{Limit: 2})
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.Equal(t, 1, page[0].ID)
	assert.Equal(t, 2, page[1].ID)

	page, err = repo.FindAll(t.Context(), port.ListParams{Limit: 2, AfterID: 2})
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, 3, page[0].ID)
}

func TestApartmentRepository_FindByBuildingID(t *testing.T) {
	buildings, repo := newRepos(t)
	b1 := mustCreateBuilding(t, buildings, "B1")
	b2 := mustCreateBuilding(t, buildings, "B2")
	empty := mustCreateBuilding(t, buildings, "Empty")
	mustCreateApartment(t, repo, b1.ID, "1")
	mustCreateApartment(t, repo, b2.ID, "1")
	mustCreateApartment(t, repo, b1.ID, "2")

	got, err := repo.FindByBuildingID(t.Context(), b1.ID, port.ListParams{})
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, a := range got {
		assert.Equal(t, b1.ID, a.BuildingID)
	}

	got, err = repo.FindByBuildingID(t.Context(), b1.ID, port.ListParams{Limit: 1})
	require.NoError(t, err)
	assert.Len(t, got, 1)

	got, err = repo.FindByBuildingID(t.Context(), empty.ID, port.ListParams{})
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = repo.FindByBuildingID(t.Context(), 999, port.ListParams{})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestApartmentRepository_Delete(t *testing.T) {
	buildings, repo := newRepos(t)
	b := mustCreateBuilding(t, buildings, "B")
	a := mustCreateApartment(t, repo, b.ID, "1")

	require.NoError(t, repo.Delete(t.Context(), a.ID))

	_, err := repo.FindByID(t.Context(), a.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	require.ErrorIs(t, repo.Delete(t.Context(), a.ID), domain.ErrNotFound)
}
