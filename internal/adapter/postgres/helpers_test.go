package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ozaitsev92/bms/internal/adapter/postgres"
)

func setupPostgresDB(t *testing.T) (*postgres.BuildingRepository, *sql.DB) {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("bms_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.PingContext(ctx))

	applyMigrations(t, ctx, db)

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("close db: %v", err)
		}
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	return postgres.NewBuildingRepo(db), db
}

func applyMigrations(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	migrationsDir := filepath.Join("..", "..", "..", "db", "migrations")

	for _, name := range []string{
		"1_create_buildings.up.sql",
		"2_create_apartments.up.sql",
	} {
		path := filepath.Join(migrationsDir, name)
		sql, err := os.ReadFile(path)
		require.NoError(t, err, "read migration %s", name)

		_, err = db.ExecContext(ctx, string(sql))
		require.NoError(t, err, "apply migration %s", name)
	}
}
