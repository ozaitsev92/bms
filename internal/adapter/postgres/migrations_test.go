package postgres_test

import (
	"context"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/require"
)

// TestMigrations_UpDownUp checks that every migration is reversible, on a
// separate database so the shared test schema is not disturbed.
func TestMigrations_UpDownUp(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: requires Docker")
	}

	_, err := testDB.ExecContext(t.Context(), "CREATE DATABASE migrations_check")
	require.NoError(t, err)
	t.Cleanup(func() {
		// t.Context() is already canceled when cleanups run.
		_, _ = testDB.ExecContext(context.Background(), "DROP DATABASE IF EXISTS migrations_check WITH (FORCE)")
	})

	mg, err := migrate.New(migrationsURL, dsnForDatabase(t, "migrations_check"))
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = mg.Close() })

	require.NoError(t, mg.Up(), "up")
	require.NoError(t, mg.Down(), "down")
	require.NoError(t, mg.Up(), "up again")

	_, dirty, err := mg.Version()
	require.NoError(t, err)
	require.False(t, dirty)
}
