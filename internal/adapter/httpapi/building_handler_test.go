package httpapi_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

func TestBuildingHandler_List_DefaultsAndParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		wantP    port.ListParams
		wantApts bool
	}{
		{name: "defaults", query: "", wantP: port.ListParams{Limit: 50}},
		{name: "explicit page", query: "?limit=10&after_id=5", wantP: port.ListParams{Limit: 10, AfterID: 5}},
		{name: "max limit", query: "?limit=200", wantP: port.ListParams{Limit: 200}},
		{name: "with apartments", query: "?with_apartments=true", wantP: port.ListParams{Limit: 50}, wantApts: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var gotP port.ListParams
			var gotApts bool
			s := newServer(t, &fakeBuildingSvc{
				list: func(_ context.Context, p port.ListParams, w bool) ([]domain.Building, error) {
					gotP, gotApts = p, w
					return []domain.Building{{ID: 1, Name: "A", Address: "1"}}, nil
				},
			}, nil, nil)

			resp := s.do(t, "GET", "/buildings"+tt.query, "")
			require.Equal(t, 200, resp.status, resp.body)
			assert.Equal(t, tt.wantP, gotP)
			assert.Equal(t, tt.wantApts, gotApts)
			assert.Len(t, decode[[]map[string]any](t, resp.body), 1)
		})
	}
}

func TestBuildingHandler_List_InvalidParams_Returns400(t *testing.T) {
	t.Parallel()
	for _, q := range []string{"?limit=0", "?limit=201", "?limit=abc", "?after_id=-1", "?after_id=x"} {
		t.Run(q, func(t *testing.T) {
			t.Parallel()
			s := newServer(t, nil, nil, nil)
			resp := s.do(t, "GET", "/buildings"+q, "")
			assert.Equal(t, 400, resp.status, resp.body)
		})
	}
}

func TestBuildingHandler_GetByID(t *testing.T) {
	t.Parallel()
	s := newServer(t, &fakeBuildingSvc{
		get: func(_ context.Context, id int) (*domain.Building, error) {
			if id != 1 {
				return nil, domain.ErrNotFound
			}
			return &domain.Building{ID: 1, Name: "A", Address: "1", Apartments: []domain.Apartment{}}, nil
		},
	}, nil, nil)

	resp := s.do(t, "GET", "/buildings/1", "")
	require.Equal(t, 200, resp.status)
	body := decode[map[string]any](t, resp.body)
	assert.Equal(t, "A", body["name"])
	assert.Equal(t, []any{}, body["apartments"], "loaded-but-empty apartments render as []")
	assert.Contains(t, body, "created_at")

	resp = s.do(t, "GET", "/buildings/2", "")
	assert.Equal(t, 404, resp.status)
	assert.Equal(t, "not found", decode[errorBody](t, resp.body).Error)
}

func TestBuildingHandler_InvalidPathID_Returns400(t *testing.T) {
	t.Parallel()
	s := newServer(t, nil, nil, nil)
	for _, id := range []string{"abc", "0", "-1", "99999999999999999999"} {
		for _, method := range []string{"GET", "DELETE"} {
			resp := s.do(t, method, "/buildings/"+id, "")
			assert.Equal(t, 400, resp.status, "%s /buildings/%s", method, id)
		}
	}
}

func TestBuildingHandler_Upsert_Created_Returns201WithLocation(t *testing.T) {
	t.Parallel()
	var got domain.Building
	s := newServer(t, &fakeBuildingSvc{
		upsert: func(_ context.Context, b domain.Building) (*domain.Building, bool, error) {
			got = b
			b.ID = 7
			return &b, true, nil
		},
	}, nil, nil)

	resp := s.do(t, "POST", "/buildings", `{"name":"Tower","address":"1 St"}`)
	require.Equal(t, 201, resp.status, resp.body)
	assert.Equal(t, "/buildings/7", resp.header.Get("Location"))
	assert.Equal(t, domain.Building{Name: "Tower", Address: "1 St"}, got)
	assert.NotContains(t, resp.body, "apartments", "apartments are omitted when not loaded")
}

func TestBuildingHandler_Upsert_Updated_Returns200(t *testing.T) {
	t.Parallel()
	s := newServer(t, &fakeBuildingSvc{
		upsert: func(_ context.Context, b domain.Building) (*domain.Building, bool, error) {
			b.ID = 1
			return &b, false, nil
		},
	}, nil, nil)

	for _, body := range []string{`{"name":"Tower","address":"2 St"}`, `{"id":1,"name":"Tower","address":"2 St"}`} {
		resp := s.do(t, "POST", "/buildings", body)
		assert.Equal(t, 200, resp.status, resp.body)
		assert.Empty(t, resp.header.Get("Location"))
	}
}

func TestBuildingHandler_Upsert_BadRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		contentType string
		want        int
	}{
		{name: "wrong content type", body: `{"name":"A","address":"1"}`, contentType: "text/plain", want: 415},
		{name: "malformed json", body: `{"name":`, want: 400},
		{name: "unknown field", body: `{"name":"A","address":"1","floors":3}`, want: 400},
		{name: "trailing data", body: `{"name":"A","address":"1"}{}`, want: 400},
		{name: "wrong type", body: `{"name":1,"address":"1"}`, want: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newServer(t, nil, nil, nil) // the service must not be reached
			ct := tt.contentType
			if ct == "" {
				ct = "application/json"
			}
			resp := s.doWithType(t, "POST", "/buildings", tt.body, ct)
			assert.Equal(t, tt.want, resp.status, resp.body)
			assert.NotEmpty(t, decode[errorBody](t, resp.body).Error)
		})
	}
}

func TestBuildingHandler_Delete(t *testing.T) {
	t.Parallel()
	s := newServer(t, &fakeBuildingSvc{
		del: func(_ context.Context, id int) error {
			if id != 1 {
				return domain.ErrNotFound
			}
			return nil
		},
	}, nil, nil)

	resp := s.do(t, "DELETE", "/buildings/1", "")
	assert.Equal(t, 204, resp.status)
	assert.Empty(t, resp.body)

	assert.Equal(t, 404, s.do(t, "DELETE", "/buildings/2", "").status)
}
