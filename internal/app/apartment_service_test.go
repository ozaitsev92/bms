package app_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/app"
	"github.com/ozaitsev92/bms/internal/domain"
)

type mockApartmentRepo struct {
	apartments map[int]domain.Apartment
	nextID     int
}

func newMockApartmentRepo() *mockApartmentRepo {
	return &mockApartmentRepo{
		apartments: make(map[int]domain.Apartment),
		nextID:     1,
	}
}

func (m *mockApartmentRepo) FindAll(_ context.Context) ([]domain.Apartment, error) {
	out := make([]domain.Apartment, 0, len(m.apartments))
	for _, a := range m.apartments {
		out = append(out, a)
	}
	return out, nil
}

func (m *mockApartmentRepo) FindByID(_ context.Context, id int) (*domain.Apartment, error) {
	a, ok := m.apartments[id]
	if !ok {
		return nil, app.ErrNotFound
	}
	return &a, nil
}

func (m *mockApartmentRepo) FindByBuildingID(_ context.Context, buildingID int) ([]domain.Apartment, error) {
	var out []domain.Apartment
	for _, a := range m.apartments {
		if a.BuildingID == buildingID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *mockApartmentRepo) Upsert(_ context.Context, a domain.Apartment) (*domain.Apartment, error) {
	if a.ID == 0 {
		a.ID = m.nextID
		m.nextID++
	}
	m.apartments[a.ID] = a
	return &a, nil
}

func (m *mockApartmentRepo) Delete(_ context.Context, id int) error {
	if _, ok := m.apartments[id]; !ok {
		return app.ErrNotFound
	}
	delete(m.apartments, id)
	return nil
}

type mockApartmentRepoNoRowsFind struct {
	*mockApartmentRepo
}

func (m *mockApartmentRepoNoRowsFind) FindByID(_ context.Context, _ int) (*domain.Apartment, error) {
	return nil, sql.ErrNoRows
}

type mockApartmentRepoNoRowsDelete struct {
	*mockApartmentRepo
}

func (m *mockApartmentRepoNoRowsDelete) Delete(_ context.Context, _ int) error {
	return sql.ErrNoRows
}

func TestApartmentService_Upsert(t *testing.T) {
	svc := app.NewApartmentService(newMockApartmentRepo())

	t.Run("creates valid apartment", func(t *testing.T) {
		a, err := svc.Upsert(context.Background(), domain.Apartment{
			BuildingID: 1,
			Number:     "42A",
			Floor:      3,
			SqMeters:   65,
		})
		require.NoError(t, err)
		assert.NotZero(t, a.ID)
	})

	t.Run("rejects missing building_id", func(t *testing.T) {
		_, err := svc.Upsert(context.Background(), domain.Apartment{
			Number: "1A", Floor: 1, SqMeters: 40,
		})
		assert.ErrorIs(t, err, app.ErrMissingBuildingID)
	})

	t.Run("rejects missing number", func(t *testing.T) {
		_, err := svc.Upsert(context.Background(), domain.Apartment{
			BuildingID: 1, Floor: 1, SqMeters: 40,
		})
		assert.ErrorIs(t, err, app.ErrMissingApartmentNumber)
	})

	t.Run("rejects invalid floor", func(t *testing.T) {
		_, err := svc.Upsert(context.Background(), domain.Apartment{
			BuildingID: 1, Number: "1A", Floor: 0, SqMeters: 40,
		})
		assert.ErrorIs(t, err, app.ErrInvalidFloor)
	})

	t.Run("rejects invalid sq_meters", func(t *testing.T) {
		_, err := svc.Upsert(context.Background(), domain.Apartment{
			BuildingID: 1, Number: "1A", Floor: 1, SqMeters: 0,
		})
		assert.ErrorIs(t, err, app.ErrInvalidSqMeters)
	})
}

func TestApartmentService_GetByBuildingID(t *testing.T) {
	repo := newMockApartmentRepo()
	svc := app.NewApartmentService(repo)

	for _, a := range []domain.Apartment{
		{BuildingID: 1, Number: "1A", Floor: 1, SqMeters: 50},
		{BuildingID: 1, Number: "1B", Floor: 1, SqMeters: 55},
		{BuildingID: 2, Number: "2A", Floor: 2, SqMeters: 70},
	} {
		_, err := svc.Upsert(context.Background(), a)
		require.NoError(t, err)
	}

	apts, err := svc.GetByBuildingID(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, apts, 2)
	for _, a := range apts {
		assert.Equal(t, 1, a.BuildingID)
	}
}

func TestApartmentService_GetByID(t *testing.T) {
	repo := newMockApartmentRepo()
	svc := app.NewApartmentService(repo)

	created, err := svc.Upsert(context.Background(), domain.Apartment{
		BuildingID: 1, Number: "1A", Floor: 1, SqMeters: 50,
	})
	require.NoError(t, err)

	found, err := svc.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "1A", found.Number)
}

func TestApartmentService_GetByID_NotFoundFromRepo(t *testing.T) {
	repo := &mockApartmentRepoNoRowsFind{mockApartmentRepo: newMockApartmentRepo()}
	svc := app.NewApartmentService(repo)

	_, err := svc.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, app.ErrNotFound)
}

func TestApartmentService_Delete(t *testing.T) {
	repo := newMockApartmentRepo()
	svc := app.NewApartmentService(repo)

	a, err := svc.Upsert(context.Background(), domain.Apartment{
		BuildingID: 1, Number: "1A", Floor: 1, SqMeters: 50,
	})
	require.NoError(t, err)

	err = svc.Delete(context.Background(), a.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), a.ID)
	assert.Error(t, err)
}

func TestApartmentService_Delete_NotFoundFromRepo(t *testing.T) {
	repo := &mockApartmentRepoNoRowsDelete{mockApartmentRepo: newMockApartmentRepo()}
	svc := app.NewApartmentService(repo)

	err := svc.Delete(context.Background(), 999)
	assert.ErrorIs(t, err, app.ErrNotFound)
}
