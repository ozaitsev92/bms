// Package httpapi is the HTTP (Fiber) adapter: routing, middleware, request
// decoding, DTOs and the mapping of domain errors to HTTP status codes.
package httpapi

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// Pinger reports whether a dependency (the database) is reachable.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Config holds the HTTP server limits.
type Config struct {
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	RequestTimeout time.Duration
	BodyLimit      int
}

// New builds the Fiber app with middleware and all routes registered.
func New(cfg Config, log *slog.Logger, ready Pinger, bh *BuildingHandler, ah *ApartmentHandler) *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		IdleTimeout:           cfg.IdleTimeout,
		BodyLimit:             cfg.BodyLimit,
		ErrorHandler:          ErrorHandler(log),
		DisableStartupMessage: true,
	})

	app.Use(
		requestid.New(),
		accessLog(log),
		recover.New(),
		requestTimeout(cfg.RequestTimeout),
	)

	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	app.Get("/readyz", readiness(log, ready))

	buildings := app.Group("/buildings")
	buildings.Get("/", bh.List)
	buildings.Get("/:id", bh.GetByID)
	buildings.Get("/:id/apartments", ah.ListByBuilding)
	buildings.Post("/", bh.Upsert)
	buildings.Delete("/:id", bh.Delete)

	apartments := app.Group("/apartments")
	apartments.Get("/", ah.List)
	apartments.Get("/building/:id", ah.ListByBuilding)
	apartments.Get("/:id", ah.GetByID)
	apartments.Post("/", ah.Upsert)
	apartments.Delete("/:id", ah.Delete)

	return app
}

func readiness(log *slog.Logger, p Pinger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), time.Second)
		defer cancel()
		if err := p.PingContext(ctx); err != nil {
			log.WarnContext(ctx, "readiness check failed",
				slog.String("request_id", requestID(c)),
				slog.String("error", err.Error()),
			)
			return fiber.NewError(fiber.StatusServiceUnavailable, "database unavailable")
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

func requestTimeout(d time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if d <= 0 {
			return c.Next()
		}
		ctx, cancel := context.WithTimeout(c.UserContext(), d)
		defer cancel()
		c.SetUserContext(ctx)

		return c.Next()
	}
}

func accessLog(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		if err := c.Next(); err != nil {
			if herr := c.App().ErrorHandler(c, err); herr != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		log.InfoContext(c.UserContext(), "http request",
			slog.String("request_id", requestID(c)),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", c.Response().StatusCode()),
			slog.Duration("duration", time.Since(start)),
		)

		return nil
	}
}

func requestID(c *fiber.Ctx) string {
	id, _ := c.Locals(requestid.ConfigDefault.ContextKey).(string)
	return id
}
