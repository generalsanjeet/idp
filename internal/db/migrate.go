package db

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrate runs all pending migration files from the migrations/ directory.
// It is safe to call on every startup — already-run migrations are skipped.
// The migrations directory path is relative to where the binary runs from.
func Migrate(db *sql.DB, migrationsPath string) error {
	// Create a Postgres driver instance for golang-migrate.
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create migration driver: %w", err)
	}

	// Point migrate at our SQL files.
	// "file://" prefix tells migrate to read from the filesystem.
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	// Run all pending migrations.
	// migrate.ErrNoChange means everything is already up to date — not an error.
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("could not run migrations: %w", err)
	}

	return nil
}
