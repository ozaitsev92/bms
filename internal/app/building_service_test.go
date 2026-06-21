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

type mockBuildingRepo struct {
	buildings map[int]domain.Building
	nextID    int
}

func newMockBuildingRepo() *mockBuildingRepo {
	return &mockBuildingRepo{
		buildings: make(map[int]domain.Building),
		nextID:    1,
	}
}

func (m *mockBuildingRepo) FindAll(_ context.Context, _ bool) ([]domain.Building, error) {
	out := make([]domain.Building, 0, len(m.buildings))
	for _, b := range m.buildings {
		out = append(out, b)
	}
	return out, nil
}

func (m *mockBuildingRepo) FindByID(_ context.Context, id int, _ bool) (*domain.Building, error) {
	b, ok := m.buildings[id]
	if !ok {
		return nil, app.ErrNotFound
	}
	return &b, nil
}

func (m *mockBuildingRepo) Upsert(_ context.Context, b domain.Building) (*domain.Building, error) {
	if b.ID == 0 {
		b.ID = m.nextID
		m.nextID++
	}
	m.buildings[b.ID] = b
	return &b, nil
}

func (m *mockBuildingRepo) Delete(_ context.Context, id int) error {
	if _, ok := m.buildings[id]; !ok {
		return app.ErrNotFound
	}
	delete(m.buildings, id)
	return nil
}

type mockBuildingRepoNoRowsFind struct {
	*mockBuildingRepo
}

func (m *mockBuildingRepoNoRowsFind) FindByID(_ context.Context, _ int, _ bool) (*domain.Building, error) {
	return nil, sql.ErrNoRows
}

type mockBuildingRepoNoRowsDelete struct {
	*mockBuildingRepo
}

func (m *mockBuildingRepoNoRowsDelete) Delete(_ context.Context, _ int) error {
	return sql.ErrNoRows
}

func TestBuildingService_Upsert(t *testing.T) {
	svc := app.NewBuildingService(newMockBuildingRepo())

	t.Run("creates valid building", func(t *testing.T) {
		b, err := svc.Upsert(context.Background(), domain.Building{
			Name:    "Tower A",
			Address: "123 Main St",
		})
		require.NoError(t, err)
		assert.NotZero(t, b.ID)
		assert.Equal(t, "Tower A", b.Name)
		assert.Equal(t, "123 Main St", b.Address)
	})

	t.Run("rejects missing name", func(t *testing.T) {
		_, err := svc.Upsert(context.Background(), domain.Building{
			Address: "123 Main St",
		})
		assert.ErrorIs(t, err, app.ErrMissingBuildingName)
	})

	t.Run("rejects missing address", func(t *testing.T) {
		_, err := svc.Upsert(context.Background(), domain.Building{
			Name: "Tower A",
		})
		assert.ErrorIs(t, err, app.ErrMissingBuildingAddress)
	})
}

func TestBuildingService_GetAll(t *testing.T) {
	repo := newMockBuildingRepo()
	svc := app.NewBuildingService(repo)

	for _, b := range []domain.Building{
		{Name: "Tower A", Address: "123 Main St"},
		{Name: "Tower B", Address: "456 Side Ave"},
	} {
		_, err := svc.Upsert(context.Background(), b)
		require.NoError(t, err)
	}

	buildings, err := svc.GetAll(context.Background(), false)
	require.NoError(t, err)
	assert.Len(t, buildings, 2)
}

func TestBuildingService_GetByID(t *testing.T) {
	repo := newMockBuildingRepo()
	svc := app.NewBuildingService(repo)

	created, err := svc.Upsert(context.Background(), domain.Building{
		Name: "Tower A", Address: "123 Main St",
	})
	require.NoError(t, err)

	found, err := svc.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Tower A", found.Name)
}

func TestBuildingService_GetByID_NotFoundFromRepo(t *testing.T) {
	repo := &mockBuildingRepoNoRowsFind{mockBuildingRepo: newMockBuildingRepo()}
	svc := app.NewBuildingService(repo)

	_, err := svc.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, app.ErrNotFound)
}

func TestBuildingService_Delete(t *testing.T) {
	repo := newMockBuildingRepo()
	svc := app.NewBuildingService(repo)

	b, err := svc.Upsert(context.Background(), domain.Building{
		Name:    "Tower A",
		Address: "123 Main St",
	})
	require.NoError(t, err)

	err = svc.Delete(context.Background(), b.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), b.ID)
	assert.Error(t, err)
}

func TestBuildingService_Delete_NotFoundFromRepo(t *testing.T) {
	repo := &mockBuildingRepoNoRowsDelete{mockBuildingRepo: newMockBuildingRepo()}
	svc := app.NewBuildingService(repo)

	err := svc.Delete(context.Background(), 999)
	assert.ErrorIs(t, err, app.ErrNotFound)
}
