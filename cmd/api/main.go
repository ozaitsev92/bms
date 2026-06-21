package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"

	httpadapter "github.com/ozaitsev92/bms/internal/adapter/http"
	"github.com/ozaitsev92/bms/internal/adapter/postgres"
	"github.com/ozaitsev92/bms/internal/app"
	"github.com/ozaitsev92/bms/internal/config"
	"github.com/ozaitsev92/bms/internal/logger"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	cfg := config.MustLoadPath(configPath)

	log := logger.Setup(cfg.Env)
	log.Info("starting BMS", slog.String("env", cfg.Env), slog.String("addr", cfg.HTTPServer.Address))

	db, err := openPostgres(cfg.Postgres.DSN)
	if err != nil {
		log.Error("failed to open postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Error("failed to close db", slog.String("error", closeErr.Error()))
		}
	}()
	log.Info("connected to postgres")

	// Infra
	buildingRepo := postgres.NewBuildingRepo(db)
	apartmentRepo := postgres.NewApartmentRepo(db)

	// App
	buildingSvc := app.NewBuildingService(buildingRepo)
	apartmentSvc := app.NewApartmentService(apartmentRepo)

	// HTTP
	buildingHandler := httpadapter.NewBuildingHandler(buildingSvc)
	apartmentHandler := httpadapter.NewApartmentHandler(apartmentSvc)

	f := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Error("unhandled fiber error",
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("error", err.Error()),
			)
			code := fiber.StatusInternalServerError
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	httpadapter.Register(f, buildingHandler, apartmentHandler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-quit
		log.Info("shutdown signal received", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := f.ShutdownWithContext(ctx); err != nil {
			log.Error("server forced to shut down", slog.String("error", err.Error()))
		}
	}()

	log.Info("HTTP server listening", slog.String("addr", cfg.HTTPServer.Address))

	if err := f.Listen(cfg.HTTPServer.Address); err != nil {
		log.Error("server stopped with error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("server shut down cleanly")
}

func openPostgres(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("postgres.dsn is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
