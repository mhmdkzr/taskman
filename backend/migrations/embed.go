package migrations

import "embed"

//go:embed core/*.sql
var migrationsFS embed.FS

// GetMigrationsFS returns the embedded filesystem for SQL migrations.
func GetMigrationsFS() embed.FS {
	return migrationsFS
}
