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

// EnsureColumn adds column to table if it doesn't already exist, with
// columnDef as its type/constraints (e.g. "TEXT"). schema.sql's
// CREATE TABLE IF NOT EXISTS statements only ever create a table, they never
// alter an existing one - so a column added to a table already on disk (an
// existing database from before that column existed) needs this instead.
// Safe to call on every startup: it's a no-op once the column exists.
func EnsureColumn(ctx context.Context, db *sql.DB, table, column, columnDef string) error {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close() //nolint:errcheck // read-only inspection query, nothing to roll back

	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan %s column info: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate %s columns: %w", table, err)
	}

	alterStmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnDef)
	if _, err := db.ExecContext(ctx, alterStmt); err != nil {
		return fmt.Errorf("add %s.%s column: %w", table, column, err)
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
