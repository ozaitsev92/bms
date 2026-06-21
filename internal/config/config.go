package config

import (
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds the application configuration.
type Config struct {
	Env        string           `yaml:"env" env-default:"local"`
	HTTPServer HTTPServerConfig `yaml:"http_server"`
	Postgres   PostgresConfig   `yaml:"postgres"`
}

// HTTPServerConfig holds the configuration for the HTTP server.
type HTTPServerConfig struct {
	Address                 string        `yaml:"address" env-default:"localhost:8080"`
	Timeout                 time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout             time.Duration `yaml:"idle_timeout" env-default:"60s"`
	ReadTimeout             time.Duration `yaml:"read_timeout" env-default:"10s"`
	WriteTimeout            time.Duration `yaml:"write_timeout" env-default:"10s"`
	GracefulShutdownTimeout time.Duration `yaml:"graceful_shutdown_timeout" env-default:"10s"`
}

// PostgresConfig holds the configuration for PostgreSQL connectivity.
type PostgresConfig struct {
	DSN             string `yaml:"dsn" env:"POSTGRES_DSN"`
	MigrationsTable string `yaml:"migrations_table" env-default:"schema_migrations"`
}

// MustLoadPath loads the configuration from the given path and panics on error.
func MustLoadPath(configPath string) *Config {
	if configPath == "" {
		panic("config path is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &cfg
}
