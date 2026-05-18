package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/alexpls/untils/internal/must"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type TxStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type DB interface {
	Querier
	TxStarter
}

func IsUniqueViolation(err error, constraintNames ...string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != pgerrcode.UniqueViolation {
		return false
	}
	if len(constraintNames) == 0 {
		return true
	}
	return slices.Contains(constraintNames, pgErr.ConstraintName)
}

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

func WithTxV[T any](pool DB, ctx context.Context, fn func(pgx.Tx) (T, error)) (T, error) {
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

func WithTx(pool DB, ctx context.Context, fn func(pgx.Tx) error) error {
	var dummy struct{}
	_, err := WithTxV(pool, ctx, func(tx pgx.Tx) (struct{}, error) {
		if err := fn(tx); err != nil {
			return dummy, err
		}
		return dummy, nil
	})
	return err
}
