package main

import (
	"errors"
	"log/slog"

	"github.com/alexpls/untils/internal/db/migrations"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func runMigrations(logger *slog.Logger, dbURL string) error {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		return err
	}
	defer m.Close()

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
			return nil
		}
		return err
	}

	newVersion, _, err := m.Version()
	if err != nil {
		return err
	}

	logger.Info("migrations completed", "new_version", newVersion)
	return nil
}
