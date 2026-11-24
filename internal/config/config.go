// Package config loads and validates the application configuration from a
// YAML file, with selected values overridable by environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// EnvConfigPath is the environment variable consulted when no --config flag is given.
const EnvConfigPath = "CONFIG_PATH"

// Config holds the application configuration.
type Config struct {
	Env        string           `yaml:"env" env:"APP_ENV" env-default:"local"`
	HTTPServer HTTPServerConfig `yaml:"http_server"`
	Postgres   PostgresConfig   `yaml:"postgres"`
}

// HTTPServerConfig holds the configuration for the HTTP server.
type HTTPServerConfig struct {
	Address                 string        `yaml:"address" env:"HTTP_ADDRESS" env-default:"localhost:8080"`
	RequestTimeout          time.Duration `yaml:"request_timeout" env-default:"4s"`
	IdleTimeout             time.Duration `yaml:"idle_timeout" env-default:"60s"`
	ReadTimeout             time.Duration `yaml:"read_timeout" env-default:"10s"`
	WriteTimeout            time.Duration `yaml:"write_timeout" env-default:"10s"`
	GracefulShutdownTimeout time.Duration `yaml:"graceful_shutdown_timeout" env-default:"10s"`
	BodyLimitBytes          int           `yaml:"body_limit_bytes" env-default:"1048576"`
}

// PostgresConfig holds the configuration for PostgreSQL connectivity.
type PostgresConfig struct {
	DSN             string        `yaml:"dsn" env:"POSTGRES_DSN"`
	MigrationsTable string        `yaml:"migrations_table" env-default:"schema_migrations"`
	MigrationsPath  string        `yaml:"migrations_path" env:"MIGRATIONS_PATH" env-default:"db/migrations"`
	MaxOpenConns    int           `yaml:"max_open_conns" env-default:"25"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env-default:"25"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env-default:"30m"`
}

// ResolvePath returns flagValue if set, otherwise the CONFIG_PATH variable.
func ResolvePath(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	return os.Getenv(EnvConfigPath)
}

// Load reads the YAML file at path, applies environment overrides
// (environment variables take precedence over the file) and validates it.
func Load(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is not set (use --config or %s)", EnvConfigPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}

	return &cfg, nil
}

// Validate reports every invalid setting at once.
func (c Config) Validate() error {
	var errs []error
	check := func(ok bool, msg string) {
		if !ok {
			errs = append(errs, errors.New(msg))
		}
	}

	h, p := c.HTTPServer, c.Postgres
	check(h.Address != "", "http_server.address is required")
	check(h.RequestTimeout > 0, "http_server.request_timeout must be positive")
	check(h.ReadTimeout > 0, "http_server.read_timeout must be positive")
	check(h.WriteTimeout > 0, "http_server.write_timeout must be positive")
	check(h.IdleTimeout > 0, "http_server.idle_timeout must be positive")
	check(h.GracefulShutdownTimeout > 0, "http_server.graceful_shutdown_timeout must be positive")
	check(h.BodyLimitBytes > 0, "http_server.body_limit_bytes must be positive")
	check(p.DSN != "", "postgres.dsn (or POSTGRES_DSN) is required")
	check(p.MigrationsTable != "", "postgres.migrations_table is required")
	check(p.MigrationsPath != "", "postgres.migrations_path (or MIGRATIONS_PATH) is required")
	check(p.MaxOpenConns >= 0, "postgres.max_open_conns must not be negative")
	check(p.MaxIdleConns >= 0, "postgres.max_idle_conns must not be negative")
	check(p.ConnMaxLifetime >= 0, "postgres.conn_max_lifetime must not be negative")

	return errors.Join(errs...)
}
