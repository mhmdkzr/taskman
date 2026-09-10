package merge

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string, reviewDone bool) string {
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
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.lock\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	git("add", ".gitignore")
	git("commit", "-q", "-m", "chore: init")

	now := time.Now().UTC()
	tk := task.Task{
		ID:         id,
		State:      task.StateHumanReview,
		Title:      "Test Task",
		Definition: "def",
		Git: task.Git{
			Commit: &task.GitCommit{Hash: "abc123", Message: "x", Type: "feat", At: now},
		},
	}
	if reviewDone {
		tk.State = task.StateMerge
	}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	git("add", id+".yaml")
	git("commit", "-q", "-m", "chore: seed task")
	return dir
}

func TestMerge(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	got, err := Merge(t.Context(), dir, git.NewClient(dir), Request{ID: "abc"})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got.State != task.StateCompleted {
		t.Fatalf("state = %v, want completed", got.State)
	}
}

func TestMergeRecordsBookkeepingCommit(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	git := git.NewClient(dir)
	if _, err := Merge(t.Context(), dir, git, Request{ID: "abc"}); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	clean, err := git.IsClean(t.Context())
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if !clean {
		t.Fatal("working tree not clean after Merge: want the task file committed")
	}
	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	msg := strings.TrimSpace(string(out))
	want := `chore(task): Record completion of task "Test Task" (ID: abc)`
	if msg != want {
		t.Fatalf("bookkeeping commit message = %q, want %q", msg, want)
	}
}

func TestMergeWithCommitOverride(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	newCommit := "def456"
	got, err := Merge(t.Context(), dir, git.NewClient(dir), Request{ID: "abc", Commit: newCommit})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got.Git.Commit == nil {
		t.Fatalf("git.commit is nil")
	}
	if got.Git.Commit.Hash != newCommit {
		t.Fatalf("commit hash = %v, want %v", got.Git.Commit.Hash, newCommit)
	}
}

func TestMergeRequiresReview(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	if _, err := Merge(t.Context(), dir, git.NewClient(dir), Request{ID: "abc"}); err == nil {
		t.Fatal("merge before review done: want error, got nil")
	}
}
