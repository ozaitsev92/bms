package main

import (
	"errors"
	"flag"
	"log/slog"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/ozaitsev92/bms/internal/config"
	"github.com/ozaitsev92/bms/internal/logger"
)

const migrationsPath = "./migrations"

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	cfg := config.MustLoadPath(configPath)
	log := logger.Setup(cfg.Env)

	if cfg.Postgres.DSN == "" {
		log.Error("postgres.dsn is required")
		os.Exit(1)
	}

	dsn := withQuerySeparator(cfg.Postgres.DSN) +
		"x-migrations-table=" + cfg.Postgres.MigrationsTable

	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		log.Error("failed to create migrate instance", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		_, _ = m.Close()
	}()

	log.Info("applying migrations", slog.String("path", migrationsPath))

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("no new migrations to apply")
			return
		}
		log.Error("failed to apply migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("migrations applied successfully")
}

func withQuerySeparator(dsn string) string {
	switch {
	case dsn == "":
		return dsn
	case strings.HasSuffix(dsn, "?") || strings.HasSuffix(dsn, "&"):
		return dsn
	case strings.Contains(dsn, "?"):
		return dsn + "&"
	default:
		return dsn + "?"
	}
}
