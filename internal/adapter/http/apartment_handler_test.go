package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/ozaitsev92/bms/internal/adapter/http"
	"github.com/ozaitsev92/bms/internal/app"
	"github.com/ozaitsev92/bms/internal/domain"
)

type mockApartmentService struct {
	getAll          func(ctx context.Context) ([]domain.Apartment, error)
	getByID         func(ctx context.Context, id int) (*domain.Apartment, error)
	getByBuildingID func(ctx context.Context, buildingID int) ([]domain.Apartment, error)
	upsert          func(ctx context.Context, a domain.Apartment) (*domain.Apartment, error)
	delete          func(ctx context.Context, id int) error
}

func (m *mockApartmentService) GetAll(ctx context.Context) ([]domain.Apartment, error) {
	return m.getAll(ctx)
}
func (m *mockApartmentService) GetByID(ctx context.Context, id int) (*domain.Apartment, error) {
	return m.getByID(ctx, id)
}
func (m *mockApartmentService) GetByBuildingID(ctx context.Context, buildingID int) ([]domain.Apartment, error) {
	return m.getByBuildingID(ctx, buildingID)
}
func (m *mockApartmentService) Upsert(ctx context.Context, a domain.Apartment) (*domain.Apartment, error) {
	return m.upsert(ctx, a)
}
func (m *mockApartmentService) Delete(ctx context.Context, id int) error {
	return m.delete(ctx, id)
}

func newApartmentApp(svc *mockApartmentService) *fiber.App {
	f := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})
	bh := httpadapter.NewBuildingHandler(&mockBuildingService{})
	ah := httpadapter.NewApartmentHandler(svc)
	httpadapter.Register(f, bh, ah)
	return f
}

func TestApartmentHandler_GetAll_OK(t *testing.T) {
	svc := &mockApartmentService{
		getAll: func(_ context.Context) ([]domain.Apartment, error) {
			return []domain.Apartment{
				{ID: 1, BuildingID: 1, Number: "101", Floor: 1, SqMeters: 50},
				{ID: 2, BuildingID: 1, Number: "102", Floor: 2, SqMeters: 55},
			}, nil
		},
	}

	req := httptest.NewRequest("GET", "/apartments", nil)
	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body, 2)
}

func TestApartmentHandler_GetAll_ServiceError(t *testing.T) {
	svc := &mockApartmentService{
		getAll: func(_ context.Context) ([]domain.Apartment, error) {
			return nil, errors.New("db exploded")
		},
	}

	req := httptest.NewRequest("GET", "/apartments", nil)
	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestApartmentHandler_GetByID_OK(t *testing.T) {
	svc := &mockApartmentService{
		getByID: func(_ context.Context, id int) (*domain.Apartment, error) {
			return &domain.Apartment{ID: id, BuildingID: 1, Number: "101", Floor: 1, SqMeters: 50}, nil
		},
	}

	req := httptest.NewRequest("GET", "/apartments/1", nil)
	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, float64(1), body["id"])
	assert.Equal(t, "101", body["number"])
}

func TestApartmentHandler_GetByID_NotFound(t *testing.T) {
	svc := &mockApartmentService{
		getByID: func(_ context.Context, _ int) (*domain.Apartment, error) {
			return nil, app.ErrNotFound
		},
	}

	req := httptest.NewRequest("GET", "/apartments/99", nil)
	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestApartmentHandler_GetByID_BadID(t *testing.T) {
	svc := &mockApartmentService{}

	req := httptest.NewRequest("GET", "/apartments/abc", nil)
	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestApartmentHandler_GetByBuilding_OK(t *testing.T) {
	svc := &mockApartmentService{
		getByBuildingID: func(_ context.Context, buildingID int) ([]domain.Apartment, error) {
			return []domain.Apartment{
				{ID: 1, BuildingID: buildingID, Number: "101", Floor: 1, SqMeters: 50},
			}, nil
		},
	}

	req := httptest.NewRequest("GET", "/apartments/building/1", nil)
	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body, 1)
	assert.Equal(t, float64(1), body[0]["building_id"])
}

func TestApartmentHandler_GetByBuilding_BadID(t *testing.T) {
	svc := &mockApartmentService{}

	req := httptest.NewRequest("GET", "/apartments/building/abc", nil)
	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestApartmentHandler_Upsert_Created(t *testing.T) {
	svc := &mockApartmentService{
		upsert: func(_ context.Context, a domain.Apartment) (*domain.Apartment, error) {
			a.ID = 1
			return &a, nil
		},
	}

	req := httptest.NewRequest("POST", "/apartments",
		jsonBody(t, map[string]any{
			"building_id": 1,
			"number":      "101",
			"floor":       1,
			"sq_meters":   50,
		}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, float64(1), body["id"])
	assert.Equal(t, "101", body["number"])
}

func TestApartmentHandler_Upsert_Updated(t *testing.T) {
	svc := &mockApartmentService{
		upsert: func(_ context.Context, a domain.Apartment) (*domain.Apartment, error) {
			return &a, nil
		},
	}

	req := httptest.NewRequest("POST", "/apartments",
		jsonBody(t, map[string]any{
			"id":          1,
			"building_id": 1,
			"number":      "101",
			"floor":       2,
			"sq_meters":   60,
		}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, float64(1), body["id"])
	assert.Equal(t, float64(2), body["floor"])
}

func TestApartmentHandler_Upsert_MissingBuildingID(t *testing.T) {
	svc := &mockApartmentService{
		upsert: func(_ context.Context, _ domain.Apartment) (*domain.Apartment, error) {
			return nil, app.ErrMissingBuildingID
		},
	}

	req := httptest.NewRequest("POST", "/apartments",
		jsonBody(t, map[string]any{"number": "101", "floor": 1, "sq_meters": 50}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestApartmentHandler_Upsert_InvalidBody(t *testing.T) {
	svc := &mockApartmentService{}

	req := httptest.NewRequest("POST", "/apartments", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestApartmentHandler_Delete_OK(t *testing.T) {
	svc := &mockApartmentService{
		delete: func(_ context.Context, _ int) error { return nil },
	}

	req := httptest.NewRequest("DELETE", "/apartments/1", nil)
	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestApartmentHandler_Delete_NotFound(t *testing.T) {
	svc := &mockApartmentService{
		delete: func(_ context.Context, _ int) error { return app.ErrNotFound },
	}

	req := httptest.NewRequest("DELETE", "/apartments/99", nil)
	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestApartmentHandler_Delete_BadID(t *testing.T) {
	svc := &mockApartmentService{}

	req := httptest.NewRequest("DELETE", "/apartments/abc", nil)
	resp, err := newApartmentApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
