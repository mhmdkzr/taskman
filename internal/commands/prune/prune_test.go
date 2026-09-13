package prune

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

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

// createCompleted drives a fresh task through to the completed state via the
// store's event log.
func createCompleted(t *testing.T, st *store.Store, id uuid.UUID, now time.Time) {
	t.Helper()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	events := []task.TaskEvent{
		task.SpecificationSubmitted{Specification: task.Specification{Plan: "p"}, At: now},
		task.ImplementationCompleted{
			Implementation: task.Implementation{Git: task.Git{Worktree: "/wt", Branch: "b"}},
			At:             now,
		},
		task.CommitRecorded{Commit: task.GitCommit{Hash: "c", At: now}, At: now},
		task.MergeCompleted{Merge: task.GitMerge{Target: "main", Commit: "c", At: now}, At: now},
	}
	for _, event := range events {
		if _, err := st.Append(t.Context(), id, event); err != nil {
			t.Fatalf("Append(%T) error = %v", event, err)
		}
	}
}

func createSpecified(t *testing.T, st *store.Store, id uuid.UUID, now time.Time) {
	t.Helper()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestPruneRemovesCompletedTasks(t *testing.T) {
	st := openTestStore(t)
	now := time.Now().UTC()
	completed := uuid.NewV7()
	open := uuid.NewV7()
	createCompleted(t, st, completed, now)
	createSpecified(t, st, open, now)

	got, err := Prune(t.Context(), st, Request{})
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if got.DryRun {
		t.Fatal("Prune().DryRun = true, want false")
	}
	if len(got.Pruned) != 1 || !slices.Contains(got.Pruned, completed) {
		t.Fatalf("Prune().Pruned = %v, want exactly [%s]", got.Pruned, completed)
	}
	if _, err := st.Read(t.Context(), completed); !errors.Is(err, store.ErrTaskNotFound) {
		t.Fatalf("Read(completed) error = %v, want ErrTaskNotFound", err)
	}
	if _, err := st.Read(t.Context(), open); err != nil {
		t.Fatalf("Read(open) error = %v, want the non-completed task to survive", err)
	}
}

func TestPruneDryRunKeepsTasks(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	createCompleted(t, st, id, time.Now().UTC())

	got, err := Prune(t.Context(), st, Request{DryRun: true})
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if !got.DryRun {
		t.Fatal("Prune().DryRun = false, want true")
	}
	if len(got.Pruned) != 1 || !slices.Contains(got.Pruned, id) {
		t.Fatalf("Prune().Pruned = %v, want exactly [%s]", got.Pruned, id)
	}
	if _, err := st.Read(t.Context(), id); err != nil {
		t.Fatalf("Read() error = %v, want the task to survive a dry run", err)
	}
}

func TestPruneIgnoresNonCompletedTasks(t *testing.T) {
	st := openTestStore(t)
	createSpecified(t, st, uuid.NewV7(), time.Now().UTC())

	got, err := Prune(t.Context(), st, Request{})
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if len(got.Pruned) != 0 {
		t.Fatalf("Prune().Pruned = %v, want none", got.Pruned)
	}
}
