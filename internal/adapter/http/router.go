package http

import "github.com/gofiber/fiber/v2"

// Register wires HTTP routes to building and apartment handlers.
func Register(app *fiber.App, bh *BuildingHandler, ah *ApartmentHandler) {
	api := app.Group("/")

	buildings := api.Group("/buildings")
	buildings.Get("/", bh.GetAll)
	buildings.Get("/:id", bh.GetByID)
	buildings.Post("/", bh.Upsert)
	buildings.Delete("/:id", bh.Delete)

	apartments := api.Group("/apartments")
	apartments.Get("/", ah.GetAll)
	apartments.Get("/building/:buildingId", ah.GetByBuilding)
	apartments.Get("/:id", ah.GetByID)
	apartments.Post("/", ah.Upsert)
	apartments.Delete("/:id", ah.Delete)
}
