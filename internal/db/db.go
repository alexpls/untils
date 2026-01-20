package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alexpls/untils/internal/must"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(url string, logger *slog.Logger) (pool *pgxpool.Pool, closer func()) {
	c := context.Background()

	config := must.NoErrVal(pgxpool.ParseConfig(url))
	config.ConnConfig.Tracer = loggingTracer{logger: logger}

	pool = must.NoErrVal(pgxpool.NewWithConfig(c, config))
	must.NoErr(pool.Ping(c))

	closer = func() {
		pool.Close()
	}

	return pool, closer
}

func WithTxV[T any](pool *pgxpool.Pool, ctx context.Context, fn func(pgx.Tx) (T, error)) (T, error) {
	var zero T

	tx, err := pool.Begin(ctx)
	if err != nil {
		return zero, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	res, err := fn(tx)
	if err != nil {
		return zero, err
	}

	if err = tx.Commit(ctx); err != nil {
		return zero, fmt.Errorf("committing transaction: %w", err)
	}

	return res, nil
}

func WithTx(pool *pgxpool.Pool, ctx context.Context, fn func(pgx.Tx) error) error {
	var dummy struct{}
	_, err := WithTxV(pool, ctx, func(tx pgx.Tx) (struct{}, error) {
		if err := fn(tx); err != nil {
			return dummy, err
		}
		return dummy, nil
	})
	return err
}
