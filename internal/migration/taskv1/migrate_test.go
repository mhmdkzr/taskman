package taskv1

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func TestMigrateDefinedTask(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	legacy := "task:\n  id: abc\n  state: created\n  definition: do work\n  status:\n    definition:\n      state: done\n    specification:\n      state: pending\n"
	if err := os.WriteFile(filepath.Join(dir, "abc.yaml"), []byte(legacy), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	result, err := Migrate(dir, false)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(result.Migrated) != 1 || result.Migrated[0].To != task.StateSpecify {
		t.Fatalf("result = %#v", result)
	}
	got, err := store.Read(dir, "abc")
	if err != nil {
		t.Fatalf("read migrated task: %v", err)
	}
	if got.State != task.StateSpecify {
		t.Fatalf("state = %q", got.State)
	}
}

func TestMigrateDryRunDoesNotWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "abc.yaml")
	legacy := []byte("task:\n  id: abc\n  state: created\n  definition: do work\n")
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Migrate(dir, true); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, legacy) {
		t.Fatal("dry-run changed file")
	}
}

func TestMigrateSkipsCurrentTask(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	current := task.Task{ID: "abc", State: task.StateSpecify, Definition: "do work"}
	if err := store.Write(dir, current); err != nil {
		t.Fatalf("write current task: %v", err)
	}
	path := filepath.Join(dir, "abc.yaml")
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read current task: %v", err)
	}

	result, err := Migrate(dir, false)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(result.Migrated) != 0 || len(result.Skipped) != 1 || result.Skipped[0] != "abc" {
		t.Fatalf("result = %#v, want current task skipped", result)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read task after migration: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("migration rewrote current task")
	}
}
