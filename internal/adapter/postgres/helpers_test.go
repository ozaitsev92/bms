package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ozaitsev92/bms/internal/adapter/postgres"
	"github.com/ozaitsev92/bms/internal/domain"
)

// These are integration tests: they need Docker and are skipped with -short.

const migrationsURL = "file://../../../db/migrations"

var (
	testDB  *sql.DB
	testDSN string
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}

	code, err := runWithPostgres(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "postgres test setup:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

// runWithPostgres starts one container for the whole package, applies the real
// migrations through golang-migrate and runs the tests against it.
func runWithPostgres(m *testing.M) (int, error) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("bms_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return 0, fmt.Errorf("start container: %w", err)
	}
	defer func() { _ = container.Terminate(context.Background()) }()

	testDSN, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return 0, fmt.Errorf("connection string: %w", err)
	}

	if err := migrateUp(testDSN); err != nil {
		return 0, err
	}

	testDB, err = sql.Open("postgres", testDSN)
	if err != nil {
		return 0, fmt.Errorf("open db: %w", err)
	}
	defer func() { _ = testDB.Close() }()

	return m.Run(), nil
}

func migrateUp(dsn string) error {
	mg, err := migrate.New(migrationsURL, dsn)
	if err != nil {
		return fmt.Errorf("create migrate: %w", err)
	}
	defer func() { _, _ = mg.Close() }()

	if err := mg.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

// freshDB returns the shared database with all tables emptied and sequences
// reset, so every test starts from the same state regardless of order.
func freshDB(t *testing.T) *sql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: requires Docker")
	}

	_, err := testDB.ExecContext(t.Context(), "TRUNCATE apartment, building RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	return testDB
}

func dsnForDatabase(t *testing.T, name string) string {
	t.Helper()
	i := strings.LastIndex(testDSN, "/")
	j := strings.Index(testDSN, "?")
	require.True(t, i > 0 && j > i, "unexpected DSN format %q", testDSN)

	return testDSN[:i+1] + name + testDSN[j:]
}

func mustCreateBuilding(t *testing.T, repo *postgres.BuildingRepository, name string) *domain.Building {
	t.Helper()
	b, created, err := repo.Upsert(t.Context(), domain.Building{Name: name, Address: name + " street"})
	require.NoError(t, err)
	require.True(t, created)

	return b
}

func mustCreateApartment(t *testing.T, repo *postgres.ApartmentRepository, buildingID int, number string) *domain.Apartment {
	t.Helper()
	a, created, err := repo.Upsert(t.Context(), domain.Apartment{BuildingID: buildingID, Number: number, Floor: 1, SqMeters: 50})
	require.NoError(t, err)
	require.True(t, created)

	return a
}
