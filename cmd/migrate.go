package main

import (
	"context"
	"errors"
	"log/slog"
	"net/url"

	"github.com/alexpls/untils/internal/db/migrations"
	"github.com/alexpls/untils/internal/must"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func runMigrations(logger *slog.Logger, dbURL string) error {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, migrationDriverURL(dbURL))
	if err != nil {
		return err
	}
	defer m.Close() // nolint:errcheck

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return err
	}

	if dirty {
		logger.Warn("database is in dirty state", "version", version)
	}

	logger.Info("running migrations", "current_version", version)

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("no migrations to apply")
		} else {
			return err
		}
	}

	newVersion, _, err := m.Version()
	if err != nil {
		return err
	}

	logger.Info("migrations completed", "new_version", newVersion)

	return runRiverMigrations(logger, dbURL)
}

func migrationDriverURL(dbURL string) string {
	parsed := must.NoErrVal(url.Parse(dbURL))
	switch parsed.Scheme {
	case "postgres", "postgresql":
		parsed.Scheme = "pgx5"
	}
	return parsed.String()
}

func runRiverMigrations(logger *slog.Logger, dbURL string) error {
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}

	result, err := migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil)
	if err != nil {
		return err
	}
	if len(result.Versions) == 0 {
		logger.Info("no river migrations to apply")
		return nil
	}

	logger.Info("river migrations completed", "versions_applied", len(result.Versions))
	return nil
}
