package testdb_test

import (
	"fmt"
	"testing"

	"github.com/mhmdkzr/app/pkg/testdb"
	"github.com/mhmdkzr/app/pkg/testenv"
)

func TestDBStartShared(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	if err := testdb.StartShared(); err != nil {
		t.Fatalf("start shared server: %v", err)
	}
	// Calling twice is idempotent and safe.
	if err := testdb.StartShared(); err != nil {
		t.Fatalf("start shared server twice: %v", err)
	}
}

func TestDBSetupPostgresMigrates(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	if err := testdb.StartShared(); err != nil {
		t.Fatalf("start shared server: %v", err)
	}
	db := testdb.SetupPostgres(t, "migrated")

	var exists bool
	if err := db.QueryRowContext(t.Context(), `
		SELECT EXISTS (
			SELECT 1 FROM pg_catalog.pg_tables
			WHERE schemaname = 'public' AND tablename = 'audit_logs'
		)`).Scan(&exists); err != nil {
		t.Fatalf("query audit_logs table: %v", err)
	}
	if !exists {
		t.Fatal("expected public.audit_logs to exist after migrations")
	}
}

func TestDBSetupPostgresIsolation(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	dbA := testdb.SetupPostgres(t, "iso_a")
	dbB := testdb.SetupPostgres(t, "iso_b")

	if _, err := dbA.ExecContext(t.Context(), `CREATE TABLE testdb_isolation (id integer)`); err != nil {
		t.Fatalf("create table in db A: %v", err)
	}

	var existsB bool
	if err := dbB.QueryRowContext(t.Context(), `SELECT to_regclass('testdb_isolation') IS NOT NULL`).
		Scan(&existsB); err != nil {
		t.Fatalf("check table in db B: %v", err)
	}
	if existsB {
		t.Fatal("table created in db A leaked into db B")
	}

	var existsA bool
	if err := dbA.QueryRowContext(t.Context(), `SELECT to_regclass('testdb_isolation') IS NOT NULL`).
		Scan(&existsA); err != nil {
		t.Fatalf("check table in db A: %v", err)
	}
	if !existsA {
		t.Fatal("expected table created in db A to be visible")
	}
}

func TestDBSetupPostgresParallel(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	for i := range 4 {
		t.Run(fmt.Sprintf("db%d", i), func(t *testing.T) {
			t.Parallel()
			db := testdb.SetupPostgres(t, fmt.Sprintf("par_%d", i))
			if _, err := db.ExecContext(t.Context(), `CREATE TABLE par_test (id integer)`); err != nil {
				t.Fatalf("create table: %v", err)
			}
		})
	}
}
