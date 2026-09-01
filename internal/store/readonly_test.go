package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// openStore opens a migrated Store backed by a fresh temp-file database.
func openStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	if err := Migrate(st.RW()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return st
}

// TestOpenReadOnlySyncsWithWriter seeds through the read-write handle and reads
// the rows back through the read-only handle, so a read path backed by the
// read-only handle sees the same data the writer commits.
func TestOpenReadOnlySyncsWithWriter(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	sessionID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := SaveRun(ctx, st.RW(), sessionID, Run{Model: "m", Prompt: "hello"}); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	rows, err := st.RO().QueryContext(ctx, `SELECT role FROM messages`)
	if err != nil {
		t.Fatalf("read via ro: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("no rows read via ro")
	}
	var got string
	if err := rows.Scan(&got); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if got != "user" {
		t.Errorf("first message role = %q, want %q", got, "user")
	}
}

// TestOpenReadOnlyRejectsWrites proves the read-only handle is enforced by the
// storage engine, not just by tool-level checks: every write path fails even
// though the SQL is perfectly valid.
func TestOpenReadOnlyRejectsWrites(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	ro := st.RO()
	for _, q := range []string{
		`INSERT INTO sessions (id, created_at) VALUES ('x', '2026-01-01')`,
		`UPDATE sessions SET created_at = '2026-01-02'`,
		`DELETE FROM sessions`,
		`CREATE TABLE t (id INTEGER)`,
		`DROP TABLE sessions`,
	} {
		if _, err := ro.ExecContext(ctx, q); err == nil {
			t.Errorf("expected read-only error for %q", q)
		}
	}
}

func TestOpenReadOnlyMemoryNotSupported(t *testing.T) {
	if _, err := OpenReadOnly(":memory:"); err == nil {
		t.Error("expected error for :memory:")
	}
	if _, err := Open(":memory:"); err == nil {
		t.Error("expected error opening a :memory: Store")
	}
}

func TestOpenReadOnlyMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.db")
	if _, err := OpenReadOnly(path); err == nil {
		t.Error("expected error for missing database file")
	} else if !strings.Contains(err.Error(), "open read-only") {
		t.Errorf("error = %v, want it prefixed with 'open read-only'", err)
	}
}
