package testhelper

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool     *pgxpool.Pool
	poolOnce sync.Once
)

// TestDB returns a clean database pool for testing.
// Each call cleans all test tables to ensure test isolation.
// The pool is shared across tests for performance - connection cleanup
// is handled by the OS when the test process exits.
func TestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	poolOnce.Do(func() {
		pgURL := os.Getenv("PG_TEST_URL")
		if pgURL == "" {
			t.Fatal("PG_TEST_URL environment variable is not set")
		}

		var err error
		pool, err = pgxpool.New(context.Background(), pgURL)
		if err != nil {
			t.Fatalf("failed to connect to test database: %v", err)
		}

		if err = pool.Ping(context.Background()); err != nil {
			pool.Close()
			t.Fatalf("failed to ping test database: %v", err)
		}
	})

	// Clean all tables for test isolation
	cleanTables(t, pool)

	return pool
}

// cleanTables removes all data from test tables to ensure test isolation
func cleanTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	tables := []string{
		"sessions", // Add other tables here as needed
		// "users",
		// "other_table",
	}

	ctx := context.Background()
	for _, table := range tables {
		_, err := pool.Exec(ctx, "DELETE FROM "+table)
		if err != nil {
			t.Fatalf("failed to clean %s table: %v", table, err)
		}
	}
}
