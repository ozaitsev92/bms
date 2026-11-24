// Command api runs the Building Management System HTTP API.
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

	_ "github.com/lib/pq"

	"github.com/ozaitsev92/bms/internal/adapter/httpapi"
	"github.com/ozaitsev92/bms/internal/adapter/postgres"
	"github.com/ozaitsev92/bms/internal/app"
	"github.com/ozaitsev92/bms/internal/config"
	"github.com/ozaitsev92/bms/internal/logger"
)

func main() {
	configPath := flag.String("config", "", "Path to config file (default: $CONFIG_PATH)")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintln(os.Stderr, "bms api:", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(config.ResolvePath(configPath))
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.Env, os.Stdout)
	if err != nil {
		return err
	}
	log.Info("starting BMS", slog.String("env", cfg.Env), slog.String("addr", cfg.HTTPServer.Address))

	db, err := openPostgres(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("close db", slog.String("error", err.Error()))
		}
	}()
	log.Info("connected to postgres")

	buildingSvc := app.NewBuildingService(postgres.NewBuildingRepo(db))
	apartmentSvc := app.NewApartmentService(postgres.NewApartmentRepo(db))

	srv := httpapi.New(httpapi.Config{
		ReadTimeout:    cfg.HTTPServer.ReadTimeout,
		WriteTimeout:   cfg.HTTPServer.WriteTimeout,
		IdleTimeout:    cfg.HTTPServer.IdleTimeout,
		RequestTimeout: cfg.HTTPServer.RequestTimeout,
		BodyLimit:      cfg.HTTPServer.BodyLimitBytes,
	}, log, db, httpapi.NewBuildingHandler(buildingSvc), httpapi.NewApartmentHandler(apartmentSvc))

	listenErr := make(chan error, 1)
	go func() {
		log.Info("HTTP server listening", slog.String("addr", cfg.HTTPServer.Address))
		listenErr <- srv.Listen(cfg.HTTPServer.Address)
	}()

	select {
	case err := <-listenErr:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTPServer.GracefulShutdownTimeout)
	defer cancel()
	if err := srv.ShutdownWithContext(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	log.Info("server shut down cleanly")

	return nil
}

func openPostgres(ctx context.Context, cfg config.PostgresConfig) (*sql.DB, error) {
	if cfg.DSN == "" {
		return nil, errors.New("postgres dsn is required")
	}

	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
