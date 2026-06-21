package http

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/ozaitsev92/bms/internal/app"
	"github.com/ozaitsev92/bms/internal/domain"
)

type apartmentService interface {
	GetAll(ctx context.Context) ([]domain.Apartment, error)
	GetByID(ctx context.Context, id int) (*domain.Apartment, error)
	GetByBuildingID(ctx context.Context, buildingID int) ([]domain.Apartment, error)
	Upsert(ctx context.Context, a domain.Apartment) (*domain.Apartment, error)
	Delete(ctx context.Context, id int) error
}

// ApartmentHandler handles apartment-related HTTP endpoints.
type ApartmentHandler struct {
	svc apartmentService
}

// NewApartmentHandler constructs an apartment HTTP handler.
func NewApartmentHandler(svc apartmentService) *ApartmentHandler {
	return &ApartmentHandler{svc: svc}
}

// GetAll handles GET /apartments requests.
func (h *ApartmentHandler) GetAll(c *fiber.Ctx) error {
	apartments, err := h.svc.GetAll(c.Context())
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return c.JSON(toApartmentResponseList(apartments))
}

// GetByID handles GET /apartments/:id requests.
func (h *ApartmentHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	a, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			return fiber.ErrNotFound
		}

		return fiber.ErrInternalServerError
	}

	return c.JSON(toApartmentResponse(*a))
}

// GetByBuilding handles GET /apartments/building/:buildingId requests.
func (h *ApartmentHandler) GetByBuilding(c *fiber.Ctx) error {
	buildingID, err := strconv.Atoi(c.Params("buildingId"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	apartments, err := h.svc.GetByBuildingID(c.Context(), buildingID)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return c.JSON(toApartmentResponseList(apartments))
}

// Upsert handles POST /apartments requests.
func (h *ApartmentHandler) Upsert(c *fiber.Ctx) error {
	var req apartmentRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}

	status := fiber.StatusCreated
	if req.ID != 0 {
		status = fiber.StatusOK
	}

	a, err := h.svc.Upsert(c.Context(), req.toDomain())
	if err != nil {
		if errors.Is(err, app.ErrValidation) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}

		return fiber.ErrInternalServerError
	}

	return c.Status(status).JSON(toApartmentResponse(*a))
}

// Delete handles DELETE /apartments/:id requests.
func (h *ApartmentHandler) Delete(c *fiber.Ctx) error {
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
