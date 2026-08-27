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
func Migrate(ctx context.Context, db *sql.DB, migrationsFS embed.FS) error {
	return MigrateSchema(ctx, db, migrationsFS, ".", "")
}

// MigrateSchema runs migrations from sourceDir in the given PostgreSQL schema.
func MigrateSchema(ctx context.Context, db *sql.DB, migrationsFS embed.FS, sourceDir, schema string) error {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database for migrations: %w", err)
	}

	// golang-migrate creates its schema_migrations table before running any
	// migration, and it refuses to create the table in a missing schema, so
	// ensure the schema exists up front when a non-default schema is used.
	if schema != "" {
		if _, err := db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+schema); err != nil {
			return fmt.Errorf("create %s schema: %w", schema, err)
		}
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{SchemaName: schema})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	sourceDriver, err := iofs.New(migrationsFS, sourceDir)
	if err != nil {
		return fmt.Errorf("failed to create iofs source driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	result := m.Up()
	if errors.Is(result, migrate.ErrNoChange) {
		result = nil
	}
	if closeErr := closeMigration(m); closeErr != nil {
		if result != nil {
			return fmt.Errorf("%w; %w", result, closeErr)
		}
		return closeErr
	}
	return result
}

// closeMigration closes the migrate instance, merging its source and database
// close errors so neither is silently dropped.
func closeMigration(m *migrate.Migrate) error {
	sourceErr, dbErr := m.Close()
	var err error
	if sourceErr != nil {
		err = fmt.Errorf("close migration source: %w", sourceErr)
	}
	if dbErr != nil {
		if err != nil {
			err = fmt.Errorf("%w; close migration database: %w", err, dbErr)
		} else {
			err = fmt.Errorf("close migration database: %w", dbErr)
		}
	}
	return err
}
