package store

import (
	"testing"
)

func TestMigrate(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, table := range []string{
		"sessions",
		"forks",
		"options",
		"tools",
		"session_tools",
		"messages",
		"recurrences",
		"scheduler",
	} {
		var n int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`,
			table,
		).Scan(&n); err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("table %s not created", table)
		}
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate again: %v", err)
	}
}
