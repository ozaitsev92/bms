package http

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/ozaitsev92/bms/internal/app"
	"github.com/ozaitsev92/bms/internal/domain"
)

type buildingService interface {
	GetAll(ctx context.Context, withApartments bool) ([]domain.Building, error)
	GetByID(ctx context.Context, id int) (*domain.Building, error)
	Upsert(ctx context.Context, b domain.Building) (*domain.Building, error)
	Delete(ctx context.Context, id int) error
}

// BuildingHandler handles building-related HTTP endpoints.
type BuildingHandler struct {
	svc buildingService
}

// NewBuildingHandler constructs a building HTTP handler.
func NewBuildingHandler(svc buildingService) *BuildingHandler {
	return &BuildingHandler{svc: svc}
}

// GetAll handles GET /buildings requests.
func (h *BuildingHandler) GetAll(c *fiber.Ctx) error {
	withApts := c.QueryBool("with_apartments", false)
	buildings, err := h.svc.GetAll(c.Context(), withApts)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return c.JSON(toBuildingResponseList(buildings))
}

// GetByID handles GET /buildings/:id requests.
func (h *BuildingHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	b, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			return fiber.ErrNotFound
		}

		return fiber.ErrInternalServerError
	}

	return c.JSON(toBuildingResponse(*b))
}

// Upsert handles POST /buildings requests.
func (h *BuildingHandler) Upsert(c *fiber.Ctx) error {
	var req buildingRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}

	status := fiber.StatusCreated
	if req.ID != 0 {
		status = fiber.StatusOK
	}

	b, err := h.svc.Upsert(c.Context(), req.toDomain())
	if err != nil {
		if errors.Is(err, app.ErrValidation) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		return fiber.ErrInternalServerError
	}

	return c.Status(status).JSON(toBuildingResponse(*b))
}

// Delete handles DELETE /buildings/:id requests.
func (h *BuildingHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	if err := h.svc.Delete(c.Context(), id); err != nil {
		if errors.Is(err, app.ErrNotFound) {
			return fiber.ErrNotFound
		}

		return fiber.ErrInternalServerError
	}

	return c.SendStatus(fiber.StatusNoContent)
}
