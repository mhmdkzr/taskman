package migrate

import (
	"context"
	"database/sql"
	"embed"
	"testing"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

//go:embed core/*.sql
var testMigrations embed.FS

func TestMigrate_AppliesMigrations(t *testing.T) {
	db, dsn, cleanup := setupPostgres(t)
	t.Cleanup(cleanup)

	if err := Migrate(context.Background(), db, testMigrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM core.migrate_test`).Scan(&count); err != nil {
		t.Fatalf("query migrated table: %v", err)
	}
	if count != 0 {
		t.Fatalf("unexpected row count: %d", count)
	}
}

func TestMigrate_ReturnsPingError(t *testing.T) {
	db, _, cleanup := setupPostgres(t)
	t.Cleanup(cleanup)

	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	if err := Migrate(context.Background(), db, testMigrations); err == nil {
		t.Fatal("expected migration error")
	}
}

func setupPostgres(t *testing.T) (*sql.DB, string, func()) {
	t.Helper()

	ctx := context.Background()
	pg, err := postgres.Run(
		ctx,
		"postgres:18-trixie",
		postgres.WithDatabase("migrate_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS core`); err != nil {
		t.Fatalf("create core schema: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		testcontainers.CleanupContainer(t, pg)
	}
	return db, dsn, cleanup
}
