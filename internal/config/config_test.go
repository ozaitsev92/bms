package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/config"
)

const validYAML = `
env: "prod"
http_server:
  address: ":9090"
  request_timeout: 3s
  read_timeout: 5s
  write_timeout: 6s
  idle_timeout: 7s
  graceful_shutdown_timeout: 8s
postgres:
  dsn: "postgres://file/bms"
  migrations_path: "db/migrations"
`

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoad_ReadsFileAndAppliesDefaults(t *testing.T) {
	cfg, err := config.Load(writeConfig(t, validYAML))
	require.NoError(t, err)

	assert.Equal(t, "prod", cfg.Env)
	assert.Equal(t, ":9090", cfg.HTTPServer.Address)
	assert.Equal(t, 5*time.Second, cfg.HTTPServer.ReadTimeout)
	assert.Equal(t, 8*time.Second, cfg.HTTPServer.GracefulShutdownTimeout)
	assert.Equal(t, 1<<20, cfg.HTTPServer.BodyLimitBytes, "default")
	assert.Equal(t, "schema_migrations", cfg.Postgres.MigrationsTable, "default")
	assert.Equal(t, 25, cfg.Postgres.MaxOpenConns, "default")
	assert.Equal(t, 30*time.Minute, cfg.Postgres.ConnMaxLifetime, "default")
}

func TestLoad_EnvironmentOverridesFile(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://env/bms")
	t.Setenv("MIGRATIONS_PATH", "/app/migrations")
	t.Setenv("APP_ENV", "local")

	cfg, err := config.Load(writeConfig(t, validYAML))
	require.NoError(t, err)
	assert.Equal(t, "postgres://env/bms", cfg.Postgres.DSN)
	assert.Equal(t, "/app/migrations", cfg.Postgres.MigrationsPath)
	assert.Equal(t, "local", cfg.Env)
}

func TestLoad_Errors(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		_, err := config.Load("")
		require.ErrorContains(t, err, "config path is not set")
	})
	t.Run("missing file", func(t *testing.T) {
		_, err := config.Load(filepath.Join(t.TempDir(), "nope.yaml"))
		require.Error(t, err)
	})
	t.Run("malformed yaml", func(t *testing.T) {
		_, err := config.Load(writeConfig(t, "http_server: [unclosed"))
		require.Error(t, err)
	})
	t.Run("invalid values reported together", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "")
		_, err := config.Load(writeConfig(t, "http_server:\n  read_timeout: -1s\n"))
		require.ErrorContains(t, err, "postgres.dsn")
		require.ErrorContains(t, err, "read_timeout must be positive")
	})
}

func TestResolvePath(t *testing.T) {
	t.Setenv(config.EnvConfigPath, "/from/env.yaml")

	assert.Equal(t, "/from/flag.yaml", config.ResolvePath("/from/flag.yaml"))
	assert.Equal(t, "/from/env.yaml", config.ResolvePath(""))
}
