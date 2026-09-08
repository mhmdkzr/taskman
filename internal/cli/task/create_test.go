package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreate(t *testing.T) {
	dir := newTestRepo(t)
	out, err := runCmd(t, dir, Create(), "--definition", "fix doc drift", "--title", "Fix Doc Drift")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(out, "Created task") {
		t.Fatalf("create output = %q", out)
	}
	id := onlyTaskID(t, dir)
	if _, err := os.Stat(filepath.Join(dir, ".worktrees", id)); err != nil {
		t.Errorf("worktree not created: %v", err)
	}
}

func TestCreateRequiresDefinition(t *testing.T) {
	dir := newTestRepo(t)
	if _, err := runCmd(t, dir, Create(), "--title", "No Definition"); err == nil {
		t.Fatal("create without --definition: want error, got nil")
	}
}

func TestCreateRefusesDirtyWorkingTree(t *testing.T) {
	dir := newTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := runCmd(t, dir, Create(), "--definition", "x"); err == nil {
		t.Fatal("create on dirty tree: want error, got nil")
	}
}
