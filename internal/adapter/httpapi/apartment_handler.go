package httpapi

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

type apartmentService interface {
	List(ctx context.Context, p port.ListParams) ([]domain.Apartment, error)
	GetByID(ctx context.Context, id int) (*domain.Apartment, error)
	ListByBuilding(ctx context.Context, buildingID int, p port.ListParams) ([]domain.Apartment, error)
	Upsert(ctx context.Context, a domain.Apartment) (*domain.Apartment, bool, error)
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

// List handles GET /apartments?limit=&after_id=.
func (h *ApartmentHandler) List(c *fiber.Ctx) error {
	p, err := listParams(c)
	if err != nil {
		return err
	}

	apartments, err := h.svc.List(c.UserContext(), p)
	if err != nil {
		return err
	}

	return c.JSON(toApartmentResponseList(apartments))
}

// GetByID handles GET /apartments/:id.
func (h *ApartmentHandler) GetByID(c *fiber.Ctx) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}

	a, err := h.svc.GetByID(c.UserContext(), id)
	if err != nil {
		return err
	}

	return c.JSON(toApartmentResponse(*a))
}

// ListByBuilding handles GET /buildings/:id/apartments (and the legacy
// GET /apartments/building/:id). Unknown buildings yield 404.
func (h *ApartmentHandler) ListByBuilding(c *fiber.Ctx) error {
	buildingID, err := pathID(c)
	if err != nil {
		return err
	}
	p, err := listParams(c)
	if err != nil {
		return err
	}

	apartments, err := h.svc.ListByBuilding(c.UserContext(), buildingID, p)
	if err != nil {
		return err
	}

	return c.JSON(toApartmentResponseList(apartments))
}

// Upsert handles POST /apartments. It responds 201 with a Location header
// when an apartment was inserted and 200 when an existing one was updated.
func (h *ApartmentHandler) Upsert(c *fiber.Ctx) error {
	var req apartmentRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}

	in := req.toDomain()
	if err := errors.Join(req.presenceErr(), in.Validate()); err != nil {
		return err
	}

	a, created, err := h.svc.Upsert(c.UserContext(), in)
	if err != nil {
		return err
	}

	status := fiber.StatusOK
	if created {
		status = fiber.StatusCreated
		c.Location("/apartments/" + strconv.Itoa(a.ID))
	}

	return c.Status(status).JSON(toApartmentResponse(*a))
}

// Delete handles DELETE /apartments/:id.
func (h *ApartmentHandler) Delete(c *fiber.Ctx) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}

	if err := h.svc.Delete(c.UserContext(), id); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
