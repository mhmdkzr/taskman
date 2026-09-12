package merged

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// commit returns ref's hash from the repository at dir.
func commit(t *testing.T, dir, ref string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "log", "-1", "--format=%H", ref).Output()
	if err != nil {
		t.Fatalf("git log %s: %v", ref, err)
	}
	return strings.TrimSpace(string(out))
}

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
	run("init", "-q", "-b", "main")
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

// taskAtMerge returns a task that has progressed through specification,
// implementation, and a recorded commit, positioned to record a merge.
func taskAtMerge(t *testing.T, st *store.Store, id uuid.UUID, now time.Time) {
	t.Helper()
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
		Implementation: task.Implementation{Git: task.Git{Worktree: "/wt", Branch: "feat"}},
		At:             now,
	}); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}
	if _, err := st.Append(
		t.Context(),
		id,
		task.CommitRecorded{Commit: task.GitCommit{Hash: "c1", At: now}, At: now},
	); err != nil {
		t.Fatalf("Append(CommitRecorded) error = %v", err)
	}
}

func TestMergedRecordsTargetBranchCommit(t *testing.T) {
	dir := newTestRepo(t)
	st := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()

	// Build a real merge: commit to a feature branch, then merge it into
	// main with --no-ff so main's head is a distinct merge commit.
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("checkout", "-q", "-b", "feat")
	run("commit", "--allow-empty", "-q", "-m", "feat: work")
	source := strings.TrimSpace(commit(t, dir, "HEAD"))
	run("checkout", "-q", "main")
	run("merge", "--no-ff", "feat", "-m", "Merge feat")
	merge := strings.TrimSpace(commit(t, dir, "main"))

	taskAtMerge(t, st, id, now)
	got, err := Merged(t.Context(), st, git.NewClient(dir), Request{ID: id, Target: "main"})
	if err != nil {
		t.Fatalf("Merged() error = %v", err)
	}
	if got.State() != task.StateCompleted {
		t.Fatalf("state = %s, want %s", got.State(), task.StateCompleted)
	}
	if got.Implementation.Git.Merge == nil || got.Implementation.Git.Merge.Target != "main" {
		t.Fatalf("Merge = %+v, want target main", got.Implementation.Git.Merge)
	}
	if got.Implementation.Git.Merge.Commit != merge {
		t.Fatalf("Merge.Commit = %s, want the merge commit on main %s", got.Implementation.Git.Merge.Commit, merge)
	}
	if got.Implementation.Git.Merge.Commit == source {
		t.Fatalf(
			"Merge.Commit = %s, want a commit distinct from the worktree's feature head",
			got.Implementation.Git.Merge.Commit,
		)
	}
}

func TestMergedRejectsMissingTarget(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := Merged(t.Context(), st, git.NewClient("."), Request{ID: id}); err == nil {
		t.Fatal("Merged() error = nil, want an error for a missing target")
	}
}
