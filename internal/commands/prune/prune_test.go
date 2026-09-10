package prune

import (
	"errors"
	"slices"
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func writeTask(t *testing.T, dir string, tk task.Task) {
	t.Helper()
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
}

func completed(id string) task.Task {
	return task.Task{ID: id, State: task.StateCompleted, Definition: "def"}
}

func active(id string, state task.State) task.Task {
	return task.Task{ID: id, State: state, Definition: "def"}
}

func TestPruneDeletesOnlyCompleted(t *testing.T) {
	dir := t.TempDir()
	writeTask(t, dir, completed("done1"))
	writeTask(t, dir, completed("done2"))
	writeTask(t, dir, active("started", task.StateImplement))
	writeTask(t, dir, active("failed", task.StateAbandoned))
	blocked := active("blocked", task.StateBlocked)
	blocked.Blocked = &task.Blocked{
		ResumeState: task.StateImplement,
		Stage:       task.StageImplementation,
		Reason:      "blocked",
	}
	writeTask(t, dir, blocked)
	writeTask(t, dir, active("created", task.StateSpecify))

	res, err := Prune(dir, Request{})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if res.Count != 2 {
		t.Fatalf("Count = %d, want 2", res.Count)
	}
	if !slices.Equal(res.Deleted, []string{"done1", "done2"}) {
		t.Fatalf("Deleted = %v, want [done1 done2]", res.Deleted)
	}
	for _, id := range []string{"done1", "done2"} {
		if _, err := store.Read(dir, id); !errors.Is(err, task.ErrTaskNotFound) {
			t.Fatalf("read %s after prune: want ErrTaskNotFound, got %v", id, err)
		}
	}
	for _, id := range []string{"started", "failed", "blocked", "created"} {
		if _, err := store.Read(dir, id); err != nil {
			t.Fatalf("read %s after prune: want present, got %v", id, err)
		}
	}
}

func TestPruneDryRunDeletesNothing(t *testing.T) {
	dir := t.TempDir()
	writeTask(t, dir, completed("done1"))
	writeTask(t, dir, active("started", task.StateImplement))

	res, err := Prune(dir, Request{DryRun: true})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if res.Count != 1 || !slices.Equal(res.Deleted, []string{"done1"}) {
		t.Fatalf("DryRun result = %+v, want 1 deleted [done1]", res)
	}
	if _, err := store.Read(dir, "done1"); err != nil {
		t.Fatalf("read done1 after dry-run: want present, got %v", err)
	}
}

func TestPruneNoCompleted(t *testing.T) {
	dir := t.TempDir()
	writeTask(t, dir, active("started", task.StateImplement))

	res, err := Prune(dir, Request{})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if res.Count != 0 || len(res.Deleted) != 0 {
		t.Fatalf("result = %+v, want empty", res)
	}
}

func TestPruneMissingTasksDir(t *testing.T) {
	dir := t.TempDir()
	res, err := Prune(dir, Request{})
	if err != nil {
		t.Fatalf("Prune on missing dir: %v", err)
	}
	if res.Count != 0 {
		t.Fatalf("Count = %d, want 0", res.Count)
	}
}
