package abandon

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string, state task.State) string {
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

	tk := task.Task{
		ID:         id,
		State:      state,
		Definition: "def",
		Status: task.Status{
			Definition:     task.StageStatus{State: task.StageDone},
			Specification:  task.StageStatus{State: task.StageDone},
			Implementation: task.StageStatus{State: task.StageDone},
		},
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	git("add", id+".yaml")
	git("commit", "-q", "-m", "chore: seed task")
	return dir
}

func TestAbandon(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateStarted)
	git := task.NewGit(dir)
	got, err := Abandon(t.Context(), dir, git, Request{ID: "abc", Reason: "no longer needed"})
	if err != nil {
		t.Fatalf("Abandon: %v", err)
	}
	if got.State != task.StateFailed {
		t.Fatalf("state = %v, want %v", got.State, task.StateFailed)
	}
	if got.FailureReason != "no longer needed" {
		t.Fatalf("failure_reason = %q, want %q", got.FailureReason, "no longer needed")
	}
}

func TestAbandonRecordsBookkeepingCommit(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateStarted)
	git := task.NewGit(dir)
	if _, err := Abandon(t.Context(), dir, git, Request{ID: "abc", Reason: "no longer needed"}); err != nil {
		t.Fatalf("Abandon: %v", err)
	}
	clean, err := git.IsClean(t.Context())
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if !clean {
		t.Fatal("working tree not clean after Abandon: want the task file committed")
	}
}

func TestAbandonPreventCompleted(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateCompleted)
	git := task.NewGit(dir)
	if _, err := Abandon(t.Context(), dir, git, Request{ID: "abc", Reason: "test reason"}); err == nil {
		t.Fatal("abandon completed task: want error, got nil")
	}
}

func TestAbandonFailedIdempotent(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateFailed)
	git := task.NewGit(dir)
	got, err := Abandon(t.Context(), dir, git, Request{ID: "abc", Reason: "updated reason"})
	if err != nil {
		t.Fatalf("abandon failed task again: %v", err)
	}
	if got.State != task.StateFailed {
		t.Fatalf("state = %v, want %v", got.State, task.StateFailed)
	}
	if got.FailureReason != "updated reason" {
		t.Fatalf("failure_reason = %q, want %q", got.FailureReason, "updated reason")
	}
}
