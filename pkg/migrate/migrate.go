package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
)

// Migrate applies the embedded schema to the SQLite database.
func Migrate(ctx context.Context, db *sql.DB, migrationsFS embed.FS) error {
	data, err := migrationsFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, string(data)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

// MigrateSchema is retained for callers that provide a schema directory. SQLite
// has no schemas, so sourceDir is used only to locate schema.sql.
func MigrateSchema(ctx context.Context, db *sql.DB, migrationsFS embed.FS, sourceDir, _ string) error {
	name := sourceDir + "/schema.sql"
	if sourceDir == "." || sourceDir == "" {
		name = "schema.sql"
	}
	data, err := migrationsFS.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, string(data)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
