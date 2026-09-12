package approved

import (
	"path/filepath"
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

func TestApproved(t *testing.T) {
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
		Implementation: task.Implementation{Review: task.ReviewConfiguration{
			Human: task.HumanReviewConfiguration{Required: true},
		}},
		At: now,
	}); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}
	if _, err := st.Append(id, task.CommitRecorded{Commit: task.GitCommit{Hash: "c", At: now}, At: now}); err != nil {
		t.Fatalf("Append(CommitRecorded) error = %v", err)
	}

	got, err := Approved(st, Request{ID: id, Comment: "lgtm"})
	if err != nil {
		t.Fatalf("Approved() error = %v", err)
	}
	if got.State() != task.StateMerge {
		t.Fatalf("state = %s, want %s", got.State(), task.StateMerge)
	}
}

func TestApprovedRejectsMissingID(t *testing.T) {
	st := openTestStore(t)
	if _, err := Approved(st, Request{}); err == nil {
		t.Fatal("Approved() error = nil, want an error for a missing id")
	}
}
