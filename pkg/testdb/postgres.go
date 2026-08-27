package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/mhmdkzr/app/migrations"
	"github.com/mhmdkzr/app/pkg/migrate"
	apppg "github.com/mhmdkzr/app/pkg/pg"
)

const (
	postgresImage = "postgres:18-trixie"

	// sharedContainerName is the fixed name used to reuse a single postgres
	// container across all test processes in a `go test` run.
	sharedContainerName = "testdb-pg"

	// templateDBName is migrated once and forked for every test database.
	templateDBName = "test_template"

	// maintenanceDB is the always-present database used to create new databases.
	maintenanceDB = "postgres"

	// templateInitLockKey serializes template creation+migration across test
	// processes via a Postgres advisory lock. Forking from the template is a read
	// and needs no lock; only the one-time migrate does.
	templateInitLockKey = 0x54454d504c // "TEMPL"
)

var (
	serverOnce sync.Once
	serverBase string // postgres://postgres:postgres@host:port/ (no database)
	errServer  error
)

// SetServer configures testdb to fork test databases from an existing,
// already-migrated server instead of starting its own container. dsn is the
// server base without a database, e.g. "postgres://postgres:postgres@host:port/",
// and must point at a server whose template database is migrated. It must be
// called before any SetupPostgres call (e.g. from a package TestMain). When
// unset, SetupPostgres starts a shared container lazily.
func SetServer(dsn string) {
	serverBase = dsn
}

// StartShared ensures the shared postgres server is running and its template is
// migrated, so the one-time cost is paid up front rather than by the first
// SetupPostgres call. It is idempotent and safe to call from a package
// TestMain before any SetupPostgres call.
func StartShared() error {
	ensureServer()
	return errServer
}

// SetupPostgres creates a temporary PostgreSQL database for testing and runs migrations.
//
// All test databases share a single postgres container reused by name across the
// test processes in a `go test` run. Migrations are applied once to a template
// database, and every call forks a fresh, fully isolated database from it via
// CREATE DATABASE ... TEMPLATE. This avoids per-test container startup and
// per-test migration runs.
func SetupPostgres(t *testing.T, suffix string) *sql.DB {
	t.Helper()
	ctx := t.Context()

	ensureServer()
	if errServer != nil {
		t.Fatalf("start shared postgres container: %v", errServer)
	}

	dbName := fmt.Sprintf("test_%s_%d", suffix, time.Now().UTC().UnixNano())
	if err := createDatabase(ctx, dbName); err != nil {
		t.Fatalf("create test database %s: %v", dbName, err)
	}

	db, err := sql.Open("postgres", dsnFor(dbName))
	if err != nil {
		t.Fatalf("open postgres db: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping postgres db: %v", err)
	}
	t.Cleanup(func() { apppg.CloseDB(db) })
	return db
}

// ensureServer resolves the shared server exactly once. It uses a server set via
// SetServer if one was configured; otherwise it starts (or reuses) the shared
// postgres container.
func ensureServer() {
	serverOnce.Do(func() {
		if serverBase != "" {
			return
		}
		serverBase, errServer = startSharedServer()
	})
}

// startSharedServer ensures the shared postgres container exists (reused by name)
// and that its template database is created and migrated exactly once. It returns
// the base DSN (no database name).
func startSharedServer() (string, error) {
	ctx := context.Background()

	container, err := postgres.Run(
		ctx,
		postgresImage,
		testcontainers.WithName(sharedContainerName),
		testcontainers.WithReuseByName(sharedContainerName),
		postgres.WithDatabase(templateDBName),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return "", fmt.Errorf("start shared postgres container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("read shared postgres container host: %w", err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return "", fmt.Errorf("read shared postgres container port: %w", err)
	}
	base := "postgres://postgres:postgres@" + net.JoinHostPort(host, port.Port()) + "/"

	if err := ensureTemplate(ctx, base); err != nil {
		return "", err
	}
	return base, nil
}

// ensureTemplate creates and migrates the template database exactly once. A
// cluster-wide advisory lock serializes the one-time migration across the
// processes sharing the container; processes that reuse an already-migrated
// template just observe it and proceed.
func ensureTemplate(ctx context.Context, base string) error {
	admin, err := sql.Open("postgres", base+maintenanceDB+"?sslmode=disable")
	if err != nil {
		return fmt.Errorf("open template maintenance db: %w", err)
	}
	defer apppg.CloseDB(admin)

	// A reused container may not be ready yet (it skips the wait strategy), so
	// wait until the server accepts connections.
	if err := waitReady(ctx, admin); err != nil {
		return err
	}

	conn, err := admin.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire template maintenance connection: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Error("close template maintenance connection", "error", closeErr)
		}
	}()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", templateInitLockKey); err != nil {
		return fmt.Errorf("acquire template init lock: %w", err)
	}
	defer func() {
		if _, unlockErr := conn.ExecContext(
			ctx,
			"SELECT pg_advisory_unlock($1)",
			templateInitLockKey,
		); unlockErr != nil {
			slog.Error("release template init lock", "error", unlockErr)
		}
	}()

	if err := ensureTemplateDB(ctx, admin); err != nil {
		return err
	}
	return migrateTemplateIfNeeded(ctx, base)
}

// ensureTemplateDB creates the template database if it does not exist yet.
func ensureTemplateDB(ctx context.Context, admin *sql.DB) error {
	var exists bool
	if err := admin.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)",
		templateDBName,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check template database existence: %w", err)
	}
	if exists {
		return nil
	}
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+pq.QuoteIdentifier(templateDBName)); err != nil {
		return fmt.Errorf("create template database: %w", err)
	}
	return nil
}

// migrateTemplateIfNeeded runs migrations against the template unless it is
// already migrated. Only the process holding the advisory lock reaches the
// migrate path, so migrations never run concurrently on the same template.
func migrateTemplateIfNeeded(ctx context.Context, base string) error {
	templateDB, err := sql.Open("postgres", base+templateDBName+"?sslmode=disable")
	if err != nil {
		return fmt.Errorf("open template database: %w", err)
	}
	defer apppg.CloseDB(templateDB)

	var migrated bool
	if err := templateDB.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_tables WHERE schemaname = 'public' AND tablename = 'audit_logs')`,
	).Scan(&migrated); err != nil {
		return fmt.Errorf("check template migration state: %w", err)
	}
	if migrated {
		return nil
	}

	if err := migrate.Migrate(ctx, templateDB, migrations.GetMigrationsFS()); err != nil {
		return fmt.Errorf("migrate template database: %w", err)
	}
	// The template must have no active connections so databases can be forked
	// from it; closing the handle releases the pooled connections.
	if closeErr := templateDB.Close(); closeErr != nil {
		return fmt.Errorf("close template database: %w", closeErr)
	}
	return nil
}

// waitReady blocks until the server accepts connections.
func waitReady(ctx context.Context, db *sql.DB) error {
	for {
		err := db.PingContext(ctx)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for shared postgres: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
			slog.Debug("waiting for shared postgres to accept connections", "error", err)
		}
	}
}

// dsnFor returns the connection string for the given database on the shared server.
func dsnFor(dbName string) string {
	return serverBase + dbName + "?sslmode=disable"
}

// createDatabase forks a new database from the migrated template. It connects to
// the maintenance database because the template must not be connected to while it
// is being forked.
func createDatabase(ctx context.Context, dbName string) error {
	admin, err := sql.Open("postgres", dsnFor(maintenanceDB))
	if err != nil {
		return fmt.Errorf("open maintenance db: %w", err)
	}
	defer apppg.CloseDB(admin)

	stmt := fmt.Sprintf(
		"CREATE DATABASE %s TEMPLATE %s",
		pq.QuoteIdentifier(dbName),
		pq.QuoteIdentifier(templateDBName),
	)
	if _, err := admin.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("create test database %s: %w", dbName, err)
	}
	return nil
}
