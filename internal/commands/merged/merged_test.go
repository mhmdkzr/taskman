package merged

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("commit", "--allow-empty", "-q", "-m", "chore: init")
	return dir
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return st
}

func TestMerged(t *testing.T) {
	dir := newTestRepo(t)
	st := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()
	if _, err := st.Create(id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := st.Append(id, task.SpecificationSubmitted{Specification: task.Specification{Plan: "p"}, At: now}); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}
	if _, err := st.Append(id, task.ImplementationCompleted{
		Implementation: task.Implementation{Git: task.Git{Worktree: dir, Branch: "b"}},
		At:             now,
	}); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}
	if _, err := st.Append(id, task.CommitRecorded{Commit: task.GitCommit{Hash: "c1", At: now}, At: now}); err != nil {
		t.Fatalf("Append(CommitRecorded) error = %v", err)
	}

	got, err := Merged(context.Background(), st, git.NewClient(dir), Request{ID: id, Target: "main"})
	if err != nil {
		t.Fatalf("Merged() error = %v", err)
	}
	if got.State() != task.StateCompleted {
		t.Fatalf("state = %s, want %s", got.State(), task.StateCompleted)
	}
	if got.Implementation.Git.Merge == nil || got.Implementation.Git.Merge.Target != "main" {
		t.Fatalf("Merge = %+v, want target main", got.Implementation.Git.Merge)
	}
}

func TestMergedRejectsMissingTarget(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := Merged(context.Background(), st, git.NewClient("."), Request{ID: id}); err == nil {
		t.Fatal("Merged() error = nil, want an error for a missing target")
	}
}
