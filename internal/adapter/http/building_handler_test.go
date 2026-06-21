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

type mockBuildingService struct {
	getAll  func(ctx context.Context, withApartments bool) ([]domain.Building, error)
	getByID func(ctx context.Context, id int) (*domain.Building, error)
	upsert  func(ctx context.Context, b domain.Building) (*domain.Building, error)
	delete  func(ctx context.Context, id int) error
}

func (m *mockBuildingService) GetAll(ctx context.Context, w bool) ([]domain.Building, error) {
	return m.getAll(ctx, w)
}
func (m *mockBuildingService) GetByID(ctx context.Context, id int) (*domain.Building, error) {
	return m.getByID(ctx, id)
}
func (m *mockBuildingService) Upsert(ctx context.Context, b domain.Building) (*domain.Building, error) {
	return m.upsert(ctx, b)
}
func (m *mockBuildingService) Delete(ctx context.Context, id int) error {
	return m.delete(ctx, id)
}

func newBuildingApp(svc *mockBuildingService) *fiber.App {
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
	bh := httpadapter.NewBuildingHandler(svc)
	ah := httpadapter.NewApartmentHandler(&mockApartmentService{})
	httpadapter.Register(f, bh, ah)
	return f
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

func TestBuildingHandler_GetAll_OK(t *testing.T) {
	svc := &mockBuildingService{
		getAll: func(_ context.Context, _ bool) ([]domain.Building, error) {
			return []domain.Building{
				{ID: 1, Name: "Tower A", Address: "123 Main St"},
				{ID: 2, Name: "Tower B", Address: "456 Side Ave"},
			}, nil
		},
	}

	req := httptest.NewRequest("GET", "/buildings", nil)
	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body, 2)
}

func TestBuildingHandler_GetAll_ServiceError(t *testing.T) {
	svc := &mockBuildingService{
		getAll: func(_ context.Context, _ bool) ([]domain.Building, error) {
			return nil, errors.New("db exploded")
		},
	}

	req := httptest.NewRequest("GET", "/buildings", nil)
	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestBuildingHandler_GetByID_OK(t *testing.T) {
	svc := &mockBuildingService{
		getByID: func(_ context.Context, id int) (*domain.Building, error) {
			return &domain.Building{ID: id, Name: "Tower A", Address: "123 Main St"}, nil
		},
	}

	req := httptest.NewRequest("GET", "/buildings/1", nil)
	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, float64(1), body["id"])
	assert.Equal(t, "Tower A", body["name"])
}

func TestBuildingHandler_GetByID_NotFound(t *testing.T) {
	svc := &mockBuildingService{
		getByID: func(_ context.Context, _ int) (*domain.Building, error) {
			return nil, app.ErrNotFound
		},
	}

	req := httptest.NewRequest("GET", "/buildings/99", nil)
	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestBuildingHandler_GetByID_BadID(t *testing.T) {
	svc := &mockBuildingService{}

	req := httptest.NewRequest("GET", "/buildings/abc", nil)
	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestBuildingHandler_Upsert_Created(t *testing.T) {
	svc := &mockBuildingService{
		upsert: func(_ context.Context, b domain.Building) (*domain.Building, error) {
			b.ID = 1
			return &b, nil
		},
	}

	req := httptest.NewRequest("POST", "/buildings",
		jsonBody(t, map[string]string{"name": "Tower A", "address": "123 Main St"}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, float64(1), body["id"])
	assert.Equal(t, "Tower A", body["name"])
}

func TestBuildingHandler_Upsert_Updated(t *testing.T) {
	svc := &mockBuildingService{
		upsert: func(_ context.Context, b domain.Building) (*domain.Building, error) {
			return &b, nil
		},
	}

	req := httptest.NewRequest("POST", "/buildings",
		jsonBody(t, map[string]any{"id": 1, "name": "Tower A", "address": "456 Updated Ave"}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, float64(1), body["id"])
	assert.Equal(t, "456 Updated Ave", body["address"])
}

func TestBuildingHandler_Upsert_MissingName(t *testing.T) {
	svc := &mockBuildingService{
		upsert: func(_ context.Context, _ domain.Building) (*domain.Building, error) {
			return nil, app.ErrMissingBuildingName
		},
	}

	req := httptest.NewRequest("POST", "/buildings",
		jsonBody(t, map[string]string{"address": "123 Main St"}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestBuildingHandler_Upsert_MissingAddress(t *testing.T) {
	svc := &mockBuildingService{
		upsert: func(_ context.Context, _ domain.Building) (*domain.Building, error) {
			return nil, app.ErrMissingBuildingAddress
		},
	}

	req := httptest.NewRequest("POST", "/buildings",
		jsonBody(t, map[string]string{"name": "Tower A"}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestBuildingHandler_Upsert_InvalidBody(t *testing.T) {
	svc := &mockBuildingService{}

	req := httptest.NewRequest("POST", "/buildings", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestBuildingHandler_Delete_OK(t *testing.T) {
	svc := &mockBuildingService{
		delete: func(_ context.Context, _ int) error { return nil },
	}

	req := httptest.NewRequest("DELETE", "/buildings/1", nil)
	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestBuildingHandler_Delete_NotFound(t *testing.T) {
	svc := &mockBuildingService{
		delete: func(_ context.Context, _ int) error { return app.ErrNotFound },
	}

	req := httptest.NewRequest("DELETE", "/buildings/99", nil)
	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestBuildingHandler_Delete_BadID(t *testing.T) {
	svc := &mockBuildingService{}

	req := httptest.NewRequest("DELETE", "/buildings/abc", nil)
	resp, err := newBuildingApp(svc).Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
