package httpapi

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

type buildingService interface {
	List(ctx context.Context, p port.ListParams, withApartments bool) ([]domain.Building, error)
	GetByID(ctx context.Context, id int) (*domain.Building, error)
	Upsert(ctx context.Context, b domain.Building) (*domain.Building, bool, error)
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

// List handles GET /buildings?limit=&after_id=&with_apartments=.
func (h *BuildingHandler) List(c *fiber.Ctx) error {
	p, err := listParams(c)
	if err != nil {
		return err
	}

	buildings, err := h.svc.List(c.UserContext(), p, c.QueryBool("with_apartments", false))
	if err != nil {
		return err
	}

	return c.JSON(toBuildingResponseList(buildings))
}

// GetByID handles GET /buildings/:id.
func (h *BuildingHandler) GetByID(c *fiber.Ctx) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}

	b, err := h.svc.GetByID(c.UserContext(), id)
	if err != nil {
		return err
	}

	return c.JSON(toBuildingResponse(*b))
}

// Upsert handles POST /buildings. It responds 201 with a Location header when
// a building was inserted and 200 when an existing one was updated.
func (h *BuildingHandler) Upsert(c *fiber.Ctx) error {
	var req buildingRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}

	b, created, err := h.svc.Upsert(c.UserContext(), req.toDomain())
	if err != nil {
		return err
	}

	status := fiber.StatusOK
	if created {
		status = fiber.StatusCreated
		c.Location("/buildings/" + strconv.Itoa(b.ID))
	}

	return c.Status(status).JSON(toBuildingResponse(*b))
}

// Delete handles DELETE /buildings/:id.
func (h *BuildingHandler) Delete(c *fiber.Ctx) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}

	if err := h.svc.Delete(c.UserContext(), id); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
