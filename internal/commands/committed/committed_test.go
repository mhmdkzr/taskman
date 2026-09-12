package committed

import (
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

func TestCommitted(t *testing.T) {
	dir := newTestRepo(t)
	st := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := st.Append(
		t.Context(),
		id,
		task.SpecificationSubmitted{Specification: task.Specification{Plan: "p"}, At: now},
	); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.ImplementationCompleted{
		Implementation: task.Implementation{Git: task.Git{Worktree: dir, Branch: "b"}},
		At:             now,
	}); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}

	got, err := Committed(t.Context(), st, git.NewClient(dir), Request{ID: id})
	if err != nil {
		t.Fatalf("Committed() error = %v", err)
	}
	if got.State() != task.StateMerge {
		t.Fatalf("state = %s, want %s", got.State(), task.StateMerge)
	}
	if len(got.Implementation.Git.Commits) != 1 || got.Implementation.Git.Commits[0].Hash == "" {
		t.Fatalf("Commits = %+v, want one commit with a hash", got.Implementation.Git.Commits)
	}
}

func TestCommittedRejectsBeforeImplementation(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, time.Now().UTC()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := Committed(t.Context(), st, git.NewClient("."), Request{ID: id}); err == nil {
		t.Fatal("Committed() error = nil, want an error before implementation is reported")
	}
}
