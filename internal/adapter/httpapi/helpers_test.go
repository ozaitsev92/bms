package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/adapter/httpapi"
	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

var errUnexpected = errors.New("unexpected call")

type fakeBuildingSvc struct {
	list   func(ctx context.Context, p port.ListParams, withApartments bool) ([]domain.Building, error)
	get    func(ctx context.Context, id int) (*domain.Building, error)
	upsert func(ctx context.Context, b domain.Building) (*domain.Building, bool, error)
	del    func(ctx context.Context, id int) error
}

func (f *fakeBuildingSvc) List(ctx context.Context, p port.ListParams, w bool) ([]domain.Building, error) {
	if f.list == nil {
		return nil, errUnexpected
	}
	return f.list(ctx, p, w)
}

func (f *fakeBuildingSvc) GetByID(ctx context.Context, id int) (*domain.Building, error) {
	if f.get == nil {
		return nil, errUnexpected
	}
	return f.get(ctx, id)
}

func (f *fakeBuildingSvc) Upsert(ctx context.Context, b domain.Building) (*domain.Building, bool, error) {
	if f.upsert == nil {
		return nil, false, errUnexpected
	}
	return f.upsert(ctx, b)
}

func (f *fakeBuildingSvc) Delete(ctx context.Context, id int) error {
	if f.del == nil {
		return errUnexpected
	}
	return f.del(ctx, id)
}

type fakeApartmentSvc struct {
	list       func(ctx context.Context, p port.ListParams) ([]domain.Apartment, error)
	get        func(ctx context.Context, id int) (*domain.Apartment, error)
	byBuilding func(ctx context.Context, buildingID int, p port.ListParams) ([]domain.Apartment, error)
	upsert     func(ctx context.Context, a domain.Apartment) (*domain.Apartment, bool, error)
	del        func(ctx context.Context, id int) error
}

func (f *fakeApartmentSvc) List(ctx context.Context, p port.ListParams) ([]domain.Apartment, error) {
	if f.list == nil {
		return nil, errUnexpected
	}
	return f.list(ctx, p)
}

func (f *fakeApartmentSvc) GetByID(ctx context.Context, id int) (*domain.Apartment, error) {
	if f.get == nil {
		return nil, errUnexpected
	}
	return f.get(ctx, id)
}

func (f *fakeApartmentSvc) ListByBuilding(ctx context.Context, id int, p port.ListParams) ([]domain.Apartment, error) {
	if f.byBuilding == nil {
		return nil, errUnexpected
	}
	return f.byBuilding(ctx, id, p)
}

func (f *fakeApartmentSvc) Upsert(ctx context.Context, a domain.Apartment) (*domain.Apartment, bool, error) {
	if f.upsert == nil {
		return nil, false, errUnexpected
	}
	return f.upsert(ctx, a)
}

func (f *fakeApartmentSvc) Delete(ctx context.Context, id int) error {
	if f.del == nil {
		return errUnexpected
	}
	return f.del(ctx, id)
}

type fakePinger struct{ err error }

func (p fakePinger) PingContext(context.Context) error { return p.err }

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type testServer struct {
	app  *fiber.App
	logs *syncBuffer
}

const testBodyLimit = 1024

func newServer(t *testing.T, bs *fakeBuildingSvc, as *fakeApartmentSvc, ping httpapi.Pinger) *testServer {
	t.Helper()
	if bs == nil {
		bs = &fakeBuildingSvc{}
	}
	if as == nil {
		as = &fakeApartmentSvc{}
	}
	if ping == nil {
		ping = fakePinger{}
	}
	logs := &syncBuffer{}
	log := slog.New(slog.NewJSONHandler(logs, nil))
	app := httpapi.New(httpapi.Config{
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		IdleTimeout:    time.Second,
		RequestTimeout: 2 * time.Second,
		BodyLimit:      testBodyLimit,
	}, log, ping, httpapi.NewBuildingHandler(bs), httpapi.NewApartmentHandler(as))

	return &testServer{app: app, logs: logs}
}

type response struct {
	status int
	header http.Header
	body   string
}

func (s *testServer) do(t *testing.T, method, path, body string) response {
	t.Helper()
	return s.doWithType(t, method, path, body, fiber.MIMEApplicationJSON)
}

func (s *testServer) doWithType(t *testing.T, method, path, body, contentType string) response {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequestWithContext(t.Context(), method, path, r)
	if body != "" {
		req.Header.Set(fiber.HeaderContentType, contentType)
	}

	resp, err := s.app.Test(req, -1)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return response{status: resp.StatusCode, header: resp.Header, body: string(b)}
}

func decode[T any](t *testing.T, body string) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal([]byte(body), &v), "body: %s", body)
	return v
}

type errorBody struct {
	Error   string   `json:"error"`
	Details []string `json:"details"`
}
