package implemented

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

func TestImplemented(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.SpecificationSubmitted{Specification: task.Specification{Plan: "p"}, At: now}); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}

	got, err := Implemented(t.Context(), st, Request{ID: id, Worktree: "/wt", Branch: "b"})
	if err != nil {
		t.Fatalf("Implemented() error = %v", err)
	}
	if got.State() != task.StateCommit {
		t.Fatalf("state = %s, want %s", got.State(), task.StateCommit)
	}
}

func TestImplementedRejectsReviewWithoutVerification(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := st.Append(t.Context(), id, task.SpecificationSubmitted{Specification: task.Specification{Plan: "p"}, At: now}); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}

	_, err := Implemented(t.Context(), st, Request{
		ID:       id,
		Worktree: "/wt",
		Branch:   "b",
		Review:   task.ReviewConfiguration{Agent: task.AgentReviewConfiguration{Required: true}},
	})
	if err == nil {
		t.Fatal("Implemented() error = nil, want an error for a review gate without verification")
	}
}

func TestImplementedRejectsMissingWorktree(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := Implemented(t.Context(), st, Request{ID: id, Branch: "b"}); err == nil {
		t.Fatal("Implemented() error = nil, want an error for a missing worktree")
	}
}
