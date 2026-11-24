package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/app"
	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

func apt(buildingID int, number string) domain.Apartment {
	return domain.Apartment{BuildingID: buildingID, Number: number, Floor: 1, SqMeters: 50}
}

func TestApartmentService_Upsert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		seed        []domain.Apartment
		in          domain.Apartment
		wantErr     []error
		wantCreated bool
	}{
		{name: "creates", in: apt(1, "101"), wantCreated: true},
		{name: "ground floor", in: domain.Apartment{BuildingID: 1, Number: "G", Floor: 0, SqMeters: 30}, wantCreated: true},
		{name: "existing number updates", seed: []domain.Apartment{apt(1, "101")}, in: apt(1, "101")},
		{name: "unknown id", in: domain.Apartment{ID: 9, BuildingID: 1, Number: "1", SqMeters: 1}, wantErr: []error{domain.ErrNotFound}},
		{name: "unknown building", in: apt(99, "1"), wantErr: []error{domain.ErrInvalidReference}},
		{
			name:    "number taken via update",
			seed:    []domain.Apartment{apt(1, "1"), apt(1, "2")},
			in:      domain.Apartment{ID: 2, BuildingID: 1, Number: "1", SqMeters: 1},
			wantErr: []error{domain.ErrConflict},
		},
		{name: "negative sq meters", in: domain.Apartment{BuildingID: 1, Number: "1", SqMeters: -1}, wantErr: []error{domain.ErrInvalidSqMeters}},
		{
			name:    "reports every violation",
			in:      domain.Apartment{},
			wantErr: []error{domain.ErrMissingBuildingID, domain.ErrMissingApartmentNumber, domain.ErrInvalidSqMeters},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newFakeApartmentRepo(1)
			for _, a := range tt.seed {
				_, _, err := repo.Upsert(t.Context(), a)
				require.NoError(t, err)
			}
			svc := app.NewApartmentService(repo)

			got, created, err := svc.Upsert(t.Context(), tt.in)
			if len(tt.wantErr) > 0 {
				for _, want := range tt.wantErr {
					require.ErrorIs(t, err, want)
				}
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCreated, created)
			assert.Equal(t, tt.in.Floor, got.Floor)
		})
	}
}

func TestApartmentService_GetByID(t *testing.T) {
	t.Parallel()
	repo := newFakeApartmentRepo(1)
	_, _, err := repo.Upsert(t.Context(), apt(1, "1"))
	require.NoError(t, err)
	svc := app.NewApartmentService(repo)

	got, err := svc.GetByID(t.Context(), 1)
	require.NoError(t, err)
	assert.Equal(t, "1", got.Number)

	_, err = svc.GetByID(t.Context(), 2)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestApartmentService_List(t *testing.T) {
	t.Parallel()
	repo := newFakeApartmentRepo(1, 2)
	for _, a := range []domain.Apartment{apt(1, "1"), apt(2, "1"), apt(1, "2")} {
		_, _, err := repo.Upsert(t.Context(), a)
		require.NoError(t, err)
	}
	svc := app.NewApartmentService(repo)

	all, err := svc.List(t.Context(), port.ListParams{})
	require.NoError(t, err)
	assert.Len(t, all, 3)

	page, err := svc.List(t.Context(), port.ListParams{Limit: 1, AfterID: 1})
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, 2, page[0].ID)
}

func TestApartmentService_ListByBuilding(t *testing.T) {
	t.Parallel()
	repo := newFakeApartmentRepo(1, 2, 3)
	for _, a := range []domain.Apartment{apt(1, "1"), apt(2, "1"), apt(1, "2")} {
		_, _, err := repo.Upsert(t.Context(), a)
		require.NoError(t, err)
	}
	svc := app.NewApartmentService(repo)

	got, err := svc.ListByBuilding(t.Context(), 1, port.ListParams{})
	require.NoError(t, err)
	assert.Len(t, got, 2)

	got, err = svc.ListByBuilding(t.Context(), 3, port.ListParams{})
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = svc.ListByBuilding(t.Context(), 99, port.ListParams{})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestApartmentService_Delete(t *testing.T) {
	t.Parallel()
	repo := newFakeApartmentRepo(1)
	_, _, err := repo.Upsert(t.Context(), apt(1, "1"))
	require.NoError(t, err)
	svc := app.NewApartmentService(repo)

	require.NoError(t, svc.Delete(t.Context(), 1))
	require.ErrorIs(t, svc.Delete(t.Context(), 1), domain.ErrNotFound)
}

func TestApartmentService_RepoErrorsArePropagated(t *testing.T) {
	t.Parallel()
	repo := newFakeApartmentRepo(1)
	repo.err = errBoom
	svc := app.NewApartmentService(repo)

	_, err := svc.List(t.Context(), port.ListParams{})
	require.ErrorIs(t, err, errBoom)
	_, err = svc.GetByID(t.Context(), 1)
	require.ErrorIs(t, err, errBoom)
	_, err = svc.ListByBuilding(t.Context(), 1, port.ListParams{})
	require.ErrorIs(t, err, errBoom)
	_, _, err = svc.Upsert(t.Context(), apt(1, "1"))
	require.ErrorIs(t, err, errBoom)
	require.ErrorIs(t, svc.Delete(t.Context(), 1), errBoom)
}
