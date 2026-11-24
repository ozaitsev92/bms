package httpapi_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/domain"
)

func sprintf(format, arg string) string { return fmt.Sprintf(format, arg) }

func TestErrorHandler_MapsErrorsToStatus(t *testing.T) {
	t.Parallel()

	dbErr := errors.New(`pq: connection refused to 10.0.0.5`)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
		wantLogged bool
	}{
		{name: "not found", err: fmt.Errorf("get building 1: %w", domain.ErrNotFound), wantStatus: 404, wantMsg: "not found"},
		{name: "conflict", err: fmt.Errorf("x: %w: unique constraint building_name_key", domain.ErrConflict), wantStatus: 409, wantMsg: "a resource with the same unique key already exists"},
		{name: "invalid reference", err: fmt.Errorf("x: %w", domain.ErrInvalidReference), wantStatus: 422, wantMsg: "referenced building does not exist"},
		{name: "validation", err: domain.ErrMissingBuildingName, wantStatus: 422, wantMsg: "validation failed"},
		{name: "timeout", err: fmt.Errorf("q: %w", context.DeadlineExceeded), wantStatus: 503, wantMsg: "request timed out", wantLogged: true},
		{name: "unknown", err: fmt.Errorf("get building 1: %w", dbErr), wantStatus: 500, wantMsg: "internal server error", wantLogged: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newServer(t, &fakeBuildingSvc{
				get: func(context.Context, int) (*domain.Building, error) { return nil, tt.err },
			}, nil, nil)

			resp := s.do(t, "GET", "/buildings/1", "")
			require.Equal(t, tt.wantStatus, resp.status)
			assert.Equal(t, tt.wantMsg, decode[errorBody](t, resp.body).Error)
			assert.NotContains(t, resp.body, "10.0.0.5", "internals must not leak to clients")
			assert.NotContains(t, resp.body, "building_name_key")

			if tt.wantLogged {
				assert.Contains(t, s.logs.String(), `"msg":"request failed"`)
				assert.Contains(t, s.logs.String(), tt.err.Error(), "the real cause is logged")
				assert.Contains(t, s.logs.String(), `"level":"ERROR"`)
			} else {
				assert.NotContains(t, s.logs.String(), `"level":"ERROR"`, "4xx is not an error-level event")
			}
		})
	}
}

func TestServer_AccessLogAndRequestID(t *testing.T) {
	t.Parallel()
	s := newServer(t, nil, nil, nil)

	resp := s.do(t, "GET", "/buildings/abc", "")
	require.Equal(t, 400, resp.status)
	id := resp.header.Get("X-Request-ID")
	require.NotEmpty(t, id)

	logs := s.logs.String()
	assert.Contains(t, logs, `"msg":"http request"`)
	assert.Contains(t, logs, `"status":400`, "the access log records the status the client received")
	assert.Contains(t, logs, id)
}

func TestServer_HealthAndReadiness(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 204, newServer(t, nil, nil, nil).do(t, "GET", "/healthz", "").status)
	assert.Equal(t, 204, newServer(t, nil, nil, fakePinger{}).do(t, "GET", "/readyz", "").status)

	s := newServer(t, nil, nil, fakePinger{err: errors.New("dial tcp: connection refused")})
	resp := s.do(t, "GET", "/readyz", "")
	assert.Equal(t, 503, resp.status)
	assert.Equal(t, "database unavailable", decode[errorBody](t, resp.body).Error)
	assert.Contains(t, s.logs.String(), "dial tcp: connection refused", "the ping error is logged")
	assert.NotContains(t, resp.body, "connection refused", "but not returned to the client")
}

func TestServer_RecoversFromPanics(t *testing.T) {
	t.Parallel()
	calls := 0
	s := newServer(t, &fakeBuildingSvc{
		get: func(context.Context, int) (*domain.Building, error) {
			calls++
			if calls == 1 {
				panic("nil map write")
			}
			return &domain.Building{ID: 1, Name: "A", Address: "1"}, nil
		},
	}, nil, nil)

	resp := s.do(t, "GET", "/buildings/1", "")
	assert.Equal(t, 500, resp.status)
	assert.Equal(t, "internal server error", decode[errorBody](t, resp.body).Error)
	assert.Contains(t, s.logs.String(), "nil map write")

	assert.Equal(t, 200, s.do(t, "GET", "/buildings/1", "").status, "server keeps serving after a panic")
}

func TestServer_RequestContextHasDeadline(t *testing.T) {
	t.Parallel()
	var deadline time.Time
	var ok bool
	s := newServer(t, &fakeBuildingSvc{
		get: func(ctx context.Context, _ int) (*domain.Building, error) {
			deadline, ok = ctx.Deadline()
			return &domain.Building{ID: 1}, nil
		},
	}, nil, nil)

	require.Equal(t, 200, s.do(t, "GET", "/buildings/1", "").status)
	require.True(t, ok, "handlers must receive a context with a deadline")
	assert.WithinDuration(t, time.Now().Add(2*time.Second), deadline, 2*time.Second)
}

func TestServer_UnknownRouteAndMethod(t *testing.T) {
	t.Parallel()
	s := newServer(t, nil, nil, nil)

	assert.Equal(t, 404, s.do(t, "GET", "/nope", "").status)
	assert.Equal(t, 405, s.do(t, "PUT", "/buildings", "").status)
}

// The body limit is enforced by fasthttp while reading the request, which
// app.Test bypasses, so this test goes through a real listener.
func TestServer_BodyLimit_Returns413(t *testing.T) {
	t.Parallel()
	s := newServer(t, nil, nil, nil)

	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = s.app.Listener(ln) }()
	t.Cleanup(func() { _ = s.app.Shutdown() })

	body := `{"name":"` + strings.Repeat("a", 2*testBodyLimit) + `","address":"x"}`
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+ln.Addr().String()+"/buildings", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}
