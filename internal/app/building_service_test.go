package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/app"
	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

func TestBuildingService_Upsert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		seed        []domain.Building
		in          domain.Building
		wantErr     error
		wantCreated bool
	}{
		{name: "creates", in: domain.Building{Name: "A", Address: "1"}, wantCreated: true},
		{name: "existing name updates", seed: []domain.Building{{Name: "A", Address: "1"}}, in: domain.Building{Name: "A", Address: "2"}},
		{name: "update by id", seed: []domain.Building{{Name: "A", Address: "1"}}, in: domain.Building{ID: 1, Name: "B", Address: "2"}},
		{name: "unknown id", in: domain.Building{ID: 9, Name: "A", Address: "1"}, wantErr: domain.ErrNotFound},
		{
			name:    "rename to taken name",
			seed:    []domain.Building{{Name: "A", Address: "1"}, {Name: "B", Address: "1"}},
			in:      domain.Building{ID: 2, Name: "A", Address: "1"},
			wantErr: domain.ErrConflict,
		},
		{name: "missing name", in: domain.Building{Address: "1"}, wantErr: domain.ErrMissingBuildingName},
		{name: "whitespace address", in: domain.Building{Name: "A", Address: " "}, wantErr: domain.ErrMissingBuildingAddress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newFakeBuildingRepo()
			for _, b := range tt.seed {
				_, _, err := repo.Upsert(t.Context(), b)
				require.NoError(t, err)
			}
			svc := app.NewBuildingService(repo)

			got, created, err := svc.Upsert(t.Context(), tt.in)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCreated, created)
			assert.Equal(t, tt.in.Name, got.Name)
			assert.Equal(t, tt.in.Address, got.Address)
		})
	}
}

func TestBuildingService_Upsert_InvalidInputDoesNotHitRepo(t *testing.T) {
	t.Parallel()
	repo := newFakeBuildingRepo()
	repo.err = errBoom // would surface if the repo were called
	svc := app.NewBuildingService(repo)

	_, _, err := svc.Upsert(t.Context(), domain.Building{})
	require.ErrorIs(t, err, domain.ErrValidation)
	assert.NotErrorIs(t, err, errBoom)
}

func TestBuildingService_GetByID(t *testing.T) {
	t.Parallel()
	repo := newFakeBuildingRepo()
	_, _, err := repo.Upsert(t.Context(), domain.Building{Name: "A", Address: "1"})
	require.NoError(t, err)
	svc := app.NewBuildingService(repo)

	got, err := svc.GetByID(t.Context(), 1)
	require.NoError(t, err)
	assert.Equal(t, "A", got.Name)

	_, err = svc.GetByID(t.Context(), 2)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestBuildingService_List(t *testing.T) {
	t.Parallel()
	repo := newFakeBuildingRepo()
	for _, n := range []string{"A", "B", "C"} {
		_, _, err := repo.Upsert(t.Context(), domain.Building{Name: n, Address: "x"})
		require.NoError(t, err)
	}
	svc := app.NewBuildingService(repo)

	got, err := svc.List(t.Context(), port.ListParams{Limit: 2, AfterID: 1}, false)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, 2, got[0].ID)
	assert.Equal(t, 3, got[1].ID)
}

func TestBuildingService_Delete(t *testing.T) {
	t.Parallel()
	repo := newFakeBuildingRepo()
	_, _, err := repo.Upsert(t.Context(), domain.Building{Name: "A", Address: "1"})
	require.NoError(t, err)
	svc := app.NewBuildingService(repo)

	require.NoError(t, svc.Delete(t.Context(), 1))
	require.ErrorIs(t, svc.Delete(t.Context(), 1), domain.ErrNotFound)
}

func TestBuildingService_RepoErrorsArePropagated(t *testing.T) {
	t.Parallel()
	repo := newFakeBuildingRepo()
	repo.err = errBoom
	svc := app.NewBuildingService(repo)

	_, err := svc.List(t.Context(), port.ListParams{}, true)
	require.ErrorIs(t, err, errBoom)
	_, err = svc.GetByID(t.Context(), 1)
	require.ErrorIs(t, err, errBoom)
	_, _, err = svc.Upsert(t.Context(), domain.Building{Name: "A", Address: "1"})
	require.ErrorIs(t, err, errBoom)
	assert.NotErrorIs(t, err, domain.ErrValidation, "repo failures must not look like validation errors")
	require.ErrorIs(t, svc.Delete(t.Context(), 1), errBoom)
}
