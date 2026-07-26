package testdb

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq" // Register the postgres database/sql driver.
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/mhmdkzr/app/migrations"
	"github.com/mhmdkzr/app/pkg/migrate"
	corepg "github.com/mhmdkzr/app/pkg/pg"
)

// SetupTestPostgres creates a temporary PostgreSQL database for testing and runs migrations.
func SetupTestPostgres(t *testing.T, suffix string) *sql.DB {
	t.Helper()

	ctx := t.Context()
	dbName := fmt.Sprintf("app_test_%s_%d", suffix, time.Now().UTC().UnixNano())
	container, err := postgres.Run(
		ctx,
		"postgres:18-trixie",
		postgres.WithDatabase(dbName),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	testcontainers.CleanupContainer(t, container)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open postgres db: %v", err)
	}
	if err := migrate.Migrate(ctx, db, migrations.GetMigrationsFS()); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close migrated postgres db: %v", err)
	}

	db, err = sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("reopen postgres db: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping postgres db: %v", err)
	}
	t.Cleanup(func() { corepg.CloseDB(db) })
	return db
}
