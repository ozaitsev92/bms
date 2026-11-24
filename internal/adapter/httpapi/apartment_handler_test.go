package httpapi_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

func TestApartmentHandler_List(t *testing.T) {
	t.Parallel()
	var gotP port.ListParams
	s := newServer(t, nil, &fakeApartmentSvc{
		list: func(_ context.Context, p port.ListParams) ([]domain.Apartment, error) {
			gotP = p
			return []domain.Apartment{{ID: 1, BuildingID: 1, Number: "1", Floor: 0, SqMeters: 10}}, nil
		},
	}, nil)

	resp := s.do(t, "GET", "/apartments?limit=5&after_id=2", "")
	require.Equal(t, 200, resp.status, resp.body)
	assert.Equal(t, port.ListParams{Limit: 5, AfterID: 2}, gotP)
	got := decode[[]map[string]any](t, resp.body)
	require.Len(t, got, 1)
	assert.InDelta(t, 0, got[0]["floor"], 0)

	assert.Equal(t, 400, s.do(t, "GET", "/apartments?limit=-1", "").status)
}

func TestApartmentHandler_GetByID(t *testing.T) {
	t.Parallel()
	s := newServer(t, nil, &fakeApartmentSvc{
		get: func(_ context.Context, id int) (*domain.Apartment, error) {
			if id != 3 {
				return nil, domain.ErrNotFound
			}
			return &domain.Apartment{ID: 3, BuildingID: 1, Number: "3", Floor: 2, SqMeters: 40}, nil
		},
	}, nil)

	resp := s.do(t, "GET", "/apartments/3", "")
	require.Equal(t, 200, resp.status)
	assert.Equal(t, "3", decode[map[string]any](t, resp.body)["number"])

	assert.Equal(t, 404, s.do(t, "GET", "/apartments/4", "").status)
	assert.Equal(t, 400, s.do(t, "GET", "/apartments/zero", "").status)
}

func TestApartmentHandler_ListByBuilding_BothRoutes(t *testing.T) {
	t.Parallel()
	s := newServer(t, nil, &fakeApartmentSvc{
		byBuilding: func(_ context.Context, id int, p port.ListParams) ([]domain.Apartment, error) {
			if id != 1 {
				return nil, domain.ErrNotFound
			}
			assert.Equal(t, 50, p.Limit)
			return []domain.Apartment{}, nil
		},
	}, nil)

	for _, prefix := range []string{"/buildings/%s/apartments", "/apartments/building/%s"} {
		resp := s.do(t, "GET", sprintf(prefix, "1"), "")
		assert.Equal(t, 200, resp.status, prefix)
		assert.JSONEq(t, `[]`, resp.body, "empty list is [] not null")

		assert.Equal(t, 404, s.do(t, "GET", sprintf(prefix, "2"), "").status, "unknown building")
		assert.Equal(t, 400, s.do(t, "GET", sprintf(prefix, "x"), "").status, "bad id")
	}
}

func TestApartmentHandler_Upsert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		created     bool
		svcErr      error
		wantStatus  int
		wantFloor   int
		wantDetails []string
	}{
		{name: "created", body: `{"building_id":1,"number":"1","floor":2,"sq_meters":50}`, created: true, wantStatus: 201, wantFloor: 2},
		{name: "ground floor", body: `{"building_id":1,"number":"G","floor":0,"sq_meters":50}`, created: true, wantStatus: 201},
		{name: "updated", body: `{"id":4,"building_id":1,"number":"1","floor":-1,"sq_meters":50}`, wantStatus: 200, wantFloor: -1},
		{
			name:        "missing floor",
			body:        `{"building_id":1,"number":"1","sq_meters":50}`,
			wantStatus:  422,
			wantDetails: []string{"floor is required"},
		},
		{
			name:        "every violation reported",
			body:        `{"number":" "}`,
			wantStatus:  422,
			wantDetails: []string{"floor is required", "building_id must be a positive integer", "apartment number is required", "sq_meters must be greater than zero"},
		},
		{name: "unknown building", body: `{"building_id":9,"number":"1","floor":1,"sq_meters":5}`, svcErr: domain.ErrInvalidReference, wantStatus: 422},
		{name: "unknown id", body: `{"id":9,"building_id":1,"number":"1","floor":1,"sq_meters":5}`, svcErr: domain.ErrNotFound, wantStatus: 404},
		{name: "number taken", body: `{"id":2,"building_id":1,"number":"1","floor":1,"sq_meters":5}`, svcErr: domain.ErrConflict, wantStatus: 409},
		{name: "unknown field", body: `{"building_id":1,"number":"1","floor":1,"sq_meters":5,"rooms":2}`, wantStatus: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got domain.Apartment
			s := newServer(t, nil, &fakeApartmentSvc{
				upsert: func(_ context.Context, a domain.Apartment) (*domain.Apartment, bool, error) {
					if tt.svcErr != nil {
						return nil, false, tt.svcErr
					}
					got = a
					if a.ID == 0 {
						a.ID = 11
					}
					return &a, tt.created, nil
				},
			}, nil)

			resp := s.do(t, "POST", "/apartments", tt.body)
			require.Equal(t, tt.wantStatus, resp.status, resp.body)

			switch {
			case tt.wantStatus == 201:
				assert.Equal(t, "/apartments/11", resp.header.Get("Location"))
				assert.Equal(t, tt.wantFloor, got.Floor)
			case tt.wantStatus == 200:
				assert.Equal(t, tt.wantFloor, got.Floor)
			case len(tt.wantDetails) > 0:
				eb := decode[errorBody](t, resp.body)
				assert.Equal(t, "validation failed", eb.Error)
				assert.ElementsMatch(t, tt.wantDetails, eb.Details)
			}
		})
	}
}

func TestApartmentHandler_Delete(t *testing.T) {
	t.Parallel()
	s := newServer(t, nil, &fakeApartmentSvc{
		del: func(_ context.Context, id int) error {
			if id != 1 {
				return domain.ErrNotFound
			}
			return nil
		},
	}, nil)

	assert.Equal(t, 204, s.do(t, "DELETE", "/apartments/1", "").status)
	assert.Equal(t, 404, s.do(t, "DELETE", "/apartments/2", "").status)
	assert.Equal(t, 400, s.do(t, "DELETE", "/apartments/-2", "").status)
}
