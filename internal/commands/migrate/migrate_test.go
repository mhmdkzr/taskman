package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func TestMigrateDryRunDoesNotRewriteLegacyTask(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "abc.yaml")
	legacy := []byte(
		"task:\n  id: abc\n  state: created\n  definition: do work\n  status:\n    specification:\n      state: pending\n",
	)
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatalf("write legacy task: %v", err)
	}

	result, err := Migrate(dir, Request{DryRun: true})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(result.Migrated) != 1 || result.Migrated[0].To != task.StateSpecify {
		t.Fatalf("migrated = %#v, want one task targeting specify", result.Migrated)
	}
	if _, err := store.Read(dir, "abc"); err == nil {
		t.Fatal("store.Read: legacy task was unexpectedly rewritten")
	}
}
