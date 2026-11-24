// Command migrator applies or rolls back database schema migrations.
//
// Usage:
//
//	migrator [--config path] [up|down|version|force N]
//
// "up" (default) applies all pending migrations, "down" rolls back one step,
// "version" prints the current version, and "force N" marks version N as
// clean after a failed (dirty) migration has been repaired by hand.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/ozaitsev92/bms/internal/config"
	"github.com/ozaitsev92/bms/internal/logger"
)

func main() {
	configPath := flag.String("config", "", "Path to config file (default: $CONFIG_PATH)")
	flag.Parse()

	if err := run(*configPath, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "bms migrator:", err)
		os.Exit(1)
	}
}

func run(configPath string, args []string) (err error) {
	cfg, err := config.Load(config.ResolvePath(configPath))
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.Env, os.Stdout)
	if err != nil {
		return err
	}

	dsn := withQuerySeparator(cfg.Postgres.DSN) + "x-migrations-table=" + cfg.Postgres.MigrationsTable

	m, err := migrate.New("file://"+cfg.Postgres.MigrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("create migrate instance (path %q): %w", cfg.Postgres.MigrationsPath, err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if closeErr := errors.Join(srcErr, dbErr); closeErr != nil && err == nil {
			err = fmt.Errorf("close migrate: %w", closeErr)
		}
	}()

	cmd := "up"
	if len(args) > 0 {
		cmd = args[0]
	}
	log.Info("running migrations", slog.String("command", cmd), slog.String("path", cfg.Postgres.MigrationsPath))

	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	case "version":
		v, dirty, verr := m.Version()
		if verr != nil && !errors.Is(verr, migrate.ErrNilVersion) {
			return fmt.Errorf("read version: %w", verr)
		}
		log.Info("current schema version", slog.Uint64("version", uint64(v)), slog.Bool("dirty", dirty))
		return nil
	case "force":
		if len(args) != 2 {
			return errors.New("usage: migrator force <version>")
		}
		v, perr := strconv.Atoi(args[1])
		if perr != nil {
			return fmt.Errorf("invalid version %q: %w", args[1], perr)
		}
		err = m.Force(v)
	default:
		return fmt.Errorf("unknown command %q (want up, down, version or force)", cmd)
	}

	switch {
	case errors.Is(err, migrate.ErrNoChange):
		log.Info("no migrations to apply")
		return nil
	case err != nil:
		return fmt.Errorf("migrate %s: %w", cmd, err)
	}

	log.Info("migrations completed")

	return nil
}

// withQuerySeparator returns dsn ready for another query parameter to be appended.
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
