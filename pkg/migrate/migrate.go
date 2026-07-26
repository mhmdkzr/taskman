package migrate

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrate runs all pending database migrations from the given embed.FS.
func Migrate(ctx context.Context, db *sql.DB, migrationsFS embed.FS) (err error) {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database for migrations: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	sourceDriver, err := iofs.New(migrationsFS, "core")
	if err != nil {
		return fmt.Errorf("failed to create iofs source driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; close migration source: %w", err, sourceErr)
			} else {
				err = fmt.Errorf("close migration source: %w", sourceErr)
			}
		}
		if dbErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; close migration database: %w", err, dbErr)
			} else {
				err = fmt.Errorf("close migration database: %w", dbErr)
			}
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
