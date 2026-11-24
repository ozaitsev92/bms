package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ozaitsev92/bms/internal/port"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 200
)

func decodeJSON(c *fiber.Ctx, v any) error {
	ct := strings.ToLower(c.Get(fiber.HeaderContentType))
	if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
		return fiber.NewError(fiber.StatusUnsupportedMediaType, "Content-Type must be application/json")
	}

	return decodeStrict(c.Body(), v)
}

func decodeStrict(body []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body: "+err.Error())
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body: must contain a single JSON object")
	}

	return nil
}

func pathID(c *fiber.Ctx) (int, error) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "id must be a positive integer")
	}

	return id, nil
}

// listParams parses ?limit= (1..200, default 50) and ?after_id= (>= 0).
func listParams(c *fiber.Ctx) (port.ListParams, error) {
	p := port.ListParams{Limit: defaultPageLimit}

	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxPageLimit {
			return p, fiber.NewError(fiber.StatusBadRequest, "limit must be an integer between 1 and "+strconv.Itoa(maxPageLimit))
		}
		p.Limit = n
	}

	if raw := c.Query("after_id"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return p, fiber.NewError(fiber.StatusBadRequest, "after_id must be a non-negative integer")
		}
		p.AfterID = n
	}

	return p, nil
}
