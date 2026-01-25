package testhelper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/alexpls/untils/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var (
	pool     *pgxpool.Pool
	poolOnce sync.Once
)

func getPool() *pgxpool.Pool {
	poolOnce.Do(func() {
		pgURL := os.Getenv("PG_TEST_URL")
		if pgURL == "" {
			panic("PG_TEST_URL environment variable is not set")
		}

		var err error
		pool, err = pgxpool.New(context.Background(), pgURL)
		if err != nil {
			panic(fmt.Sprintf("failed to connect to test database: %v", err))
		}

		if err = pool.Ping(context.Background()); err != nil {
			pool.Close()
			panic(fmt.Sprintf("failed to ping test database: %v", err))
		}
	})

	return pool
}

func TestTx(ctx context.Context, t *testing.T) db.DB {
	tx, err := getPool().Begin(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) {
			require.NoError(t, err)
		}
	})

	return tx
}
