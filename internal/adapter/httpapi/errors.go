package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ozaitsev92/bms/internal/domain"
)

// errorResponse is the single JSON error shape returned by the API.
type errorResponse struct {
	Error   string   `json:"error"`
	Details []string `json:"details,omitempty"`
}

// ErrorHandler is the only place where errors become HTTP responses.
// Handlers just return (wrapped) errors. 5xx causes are logged with the
// request ID; clients only ever see stable, non-leaky messages.
func ErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status, resp := classify(err)
		if status >= fiber.StatusInternalServerError {
			log.ErrorContext(c.UserContext(), "request failed",
				slog.String("request_id", requestID(c)),
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.Int("status", status),
				slog.String("error", err.Error()),
			)
		}

		return c.Status(status).JSON(resp)
	}
}

func classify(err error) (int, errorResponse) {
	var fe *fiber.Error
	switch {
	case errors.As(err, &fe):
		return fe.Code, errorResponse{Error: fe.Message}
	case errors.Is(err, domain.ErrValidation):
		return fiber.StatusUnprocessableEntity, errorResponse{Error: "validation failed", Details: validationDetails(err)}
	case errors.Is(err, domain.ErrInvalidReference):
		return fiber.StatusUnprocessableEntity, errorResponse{Error: "referenced building does not exist"}
	case errors.Is(err, domain.ErrNotFound):
		return fiber.StatusNotFound, errorResponse{Error: "not found"}
	case errors.Is(err, domain.ErrConflict):
		return fiber.StatusConflict, errorResponse{Error: "a resource with the same unique key already exists"}
	case errors.Is(err, context.DeadlineExceeded):
		return fiber.StatusServiceUnavailable, errorResponse{Error: "request timed out"}
	default:
		return fiber.StatusInternalServerError, errorResponse{Error: "internal server error"}
	}
}

// validationDetails extracts the human-readable message of every validation
// error in err's tree (errors.Join and %w wrapping are both supported).
func validationDetails(err error) []string {
	var out []string
	var walk func(error)
	walk = func(e error) {
		if joined, ok := e.(interface{ Unwrap() []error }); ok { //nolint:errorlint // walking the error tree itself
			for _, inner := range joined.Unwrap() {
				walk(inner)
			}
			return
		}

		next := errors.Unwrap(e)
		if next == nil {
			return
		}
		if next == domain.ErrValidation { //nolint:errorlint // a leaf wraps the sentinel directly
			out = append(out, strings.TrimPrefix(e.Error(), domain.ErrValidation.Error()+": "))
			return
		}
		walk(next)
	}
	walk(err)

	return out
}
