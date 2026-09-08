package create

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mhmdkzr/loop/internal/task"
)

func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".worktrees/\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	git("add", "README.md", ".gitignore")
	git("commit", "-q", "-m", "chore: init")
	return dir
}

func TestCreateSuccessful(t *testing.T) {
	gitDir := newTestRepo(t)
	tasksDir := filepath.Join(gitDir, ".tasks")
	worktreesDir := filepath.Join(gitDir, ".worktrees")

	got, err := Create(
		context.Background(),
		tasksDir,
		worktreesDir,
		task.NewGit(gitDir),
		Request{
			Definition: "test task",
			Title:      "Test Task",
		},
	)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID == "" {
		t.Fatalf("task id is empty")
	}
	if got.Definition != "test task" {
		t.Fatalf("definition = %q, want 'test task'", got.Definition)
	}
	if got.Title != "Test Task" {
		t.Fatalf("title = %q, want 'Test Task'", got.Title)
	}
	if got.Git.Worktree == "" {
		t.Fatalf("worktree is empty")
	}
	if got.Git.Branch == "" {
		t.Fatalf("branch is empty")
	}
	if _, err := os.Stat(filepath.Join(tasksDir, got.ID+".yaml")); err != nil {
		t.Fatalf("task file not found: %v", err)
	}
	if _, err := os.Stat(got.Git.Worktree); err != nil {
		t.Fatalf("worktree directory not found: %v", err)
	}
}

func TestCreateDirtyWorkingTree(t *testing.T) {
	gitDir := newTestRepo(t)
	tasksDir := filepath.Join(gitDir, ".tasks")
	worktreesDir := filepath.Join(gitDir, ".worktrees")

	// Write an uncommitted file to make working tree dirty
	if err := os.WriteFile(filepath.Join(gitDir, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	_, err := Create(
		context.Background(),
		tasksDir,
		worktreesDir,
		task.NewGit(gitDir),
		Request{
			Definition: "test task",
			Title:      "Test Task",
		},
	)
	if !errors.Is(err, task.ErrWorkingTreeDirty) {
		t.Fatalf("Create with dirty working tree: got %v, want ErrWorkingTreeDirty", err)
	}
}

func TestCreateEmptyDefinition(t *testing.T) {
	gitDir := newTestRepo(t)
	tasksDir := filepath.Join(gitDir, ".tasks")
	worktreesDir := filepath.Join(gitDir, ".worktrees")

	_, err := Create(
		context.Background(),
		tasksDir,
		worktreesDir,
		task.NewGit(gitDir),
		Request{
			Definition: "",
			Title:      "Test Task",
		},
	)
	if err == nil {
		t.Fatal("Create with empty definition: want error, got nil")
	}
}
