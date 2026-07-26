package pg

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // Register the postgres database/sql driver.
)

// Open opens a PostgreSQL connection using the given configuration.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
