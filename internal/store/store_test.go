package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenProvidesReadOnlyHandle(t *testing.T) {
	st, err := Open(t.Context(), filepath.Join(t.TempDir(), "loop.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if err := st.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	ctx := context.Background()
	if _, err := st.RW().ExecContext(ctx, `CREATE TABLE items (value TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := st.RW().ExecContext(ctx, `INSERT INTO items(value) VALUES ('rw')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var value string
	if err := st.RO().QueryRowContext(ctx, `SELECT value FROM items`).Scan(&value); err != nil {
		t.Fatalf("read through RO: %v", err)
	}
	if value != "rw" {
		t.Fatalf("value = %q, want %q", value, "rw")
	}
	if _, err := st.RO().ExecContext(ctx, `INSERT INTO items(value) VALUES ('ro')`); err == nil {
		t.Fatal("write through RO succeeded")
	}
}

func TestOpenReadOnlyRejectsMemoryDatabase(t *testing.T) {
	if _, err := OpenReadOnly(t.Context(), ":memory:"); err == nil {
		t.Fatal("OpenReadOnly(:memory:) succeeded")
	}
}
