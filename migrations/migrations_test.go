package migrations

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	testpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	coremigrate "github.com/mhmdkzr/app/pkg/migrate"
)

func TestMigrationsApply(t *testing.T) {
	db, dsn, cleanup := setupMigrationPostgres(t)
	t.Cleanup(cleanup)

	if err := coremigrate.Migrate(context.Background(), db, GetMigrationsFS()); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	_ = db.Close()

	reopened, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	db = reopened
	t.Cleanup(func() { _ = db.Close() })

	for _, table := range []string{"audit_logs"} {
		if !relationExists(t, db, "public", table) {
			t.Fatalf("expected %s to exist in public schema", table)
		}
	}
}

func setupMigrationPostgres(t *testing.T) (*sql.DB, string, func()) {
	t.Helper()

	ctx := t.Context()
	pg, err := testpg.Run(
		ctx,
		"postgres:18-trixie",
		testpg.WithDatabase("migrations_test"),
		testpg.WithUsername("postgres"),
		testpg.WithPassword("postgres"),
		testpg.BasicWaitStrategies(),
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
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		testcontainers.CleanupContainer(t, pg)
	}
	return db, dsn, cleanup
}

func relationExists(t *testing.T, db *sql.DB, schema, relation string) bool {
	t.Helper()

	var exists bool
	if err := db.QueryRowContext(t.Context(), `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)`, schema, relation).Scan(&exists); err != nil {
		t.Fatalf("query relation existence: %v", err)
	}
	return exists
}

func columnExists(t *testing.T, db *sql.DB, schema, table, column string) bool {
	t.Helper()

	var exists bool
	if err := db.QueryRowContext(t.Context(), `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
		)`, schema, table, column).Scan(&exists); err != nil {
		t.Fatalf("query column existence: %v", err)
	}
	return exists
}

func enumLabelExists(t *testing.T, db *sql.DB, schema, enumName, label string) bool {
	t.Helper()

	var exists bool
	if err := db.QueryRowContext(t.Context(), `
		SELECT EXISTS (
			SELECT 1
			FROM pg_enum e
			JOIN pg_type t ON t.oid = e.enumtypid
			JOIN pg_namespace n ON n.oid = t.typnamespace
			WHERE n.nspname = $1 AND t.typname = $2 AND e.enumlabel = $3
		)`, schema, enumName, label).Scan(&exists); err != nil {
		t.Fatalf("query enum label existence: %v", err)
	}
	return exists
}
