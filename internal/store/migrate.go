package store

import (
	"database/sql"

	"github.com/mhmdkzr/taskman/migrations"
)

// Migrate runs the embedded schema. The caller owns db and is responsible for
// closing it; Migrate never closes the handle.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(migrations.GetSchemaSQL())
	return err
}
