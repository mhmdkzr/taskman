package specified

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
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "tasks.db"))
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

func TestSpecified(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(
		t.Context(),
		id,
		task.TaskDefinition{Title: "t", Description: "d"},
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := Specified(t.Context(), st, Request{ID: id, Plan: "do it"})
	if err != nil {
		t.Fatalf("Specified() error = %v", err)
	}
	if got.State() != task.StateImplement {
		t.Fatalf("state = %s, want %s", got.State(), task.StateImplement)
	}
}

func TestSpecifiedRoundTripsImplementationPolicy(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(
		t.Context(),
		id,
		task.TaskDefinition{Title: "t", Description: "d"},
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := Specified(t.Context(), st, Request{
		ID:   id,
		Plan: "do it",
		Verification: task.Verification{
			Tests:   task.TestConfiguration{Unit: true},
			Linters: true,
		},
		ImplementationReview: task.ReviewConfiguration{Agent: task.AgentReviewConfiguration{Required: true}},
		Worktree:             task.WorktreePolicy{UseWorktree: true, Worktree: "/wt", Branch: "b"},
	})
	if err != nil {
		t.Fatalf("Specified() error = %v", err)
	}
	spec := got.Specification
	if !spec.Verification.Tests.Unit || !spec.Verification.Linters {
		t.Fatalf("verification = %+v, want unit and linters required", spec.Verification)
	}
	if !spec.ImplementationReview.Agent.Required {
		t.Fatalf("implementation review = %+v, want agent required", spec.ImplementationReview)
	}
	if spec.Worktree != (task.WorktreePolicy{UseWorktree: true, Worktree: "/wt", Branch: "b"}) {
		t.Fatalf("worktree = %+v, want use-worktree=/wt on b", spec.Worktree)
	}
}

func TestSpecifiedRejectsImplementationReviewWithoutVerification(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(
		t.Context(),
		id,
		task.TaskDefinition{Title: "t", Description: "d"},
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err := Specified(t.Context(), st, Request{
		ID:                   id,
		Plan:                 "do it",
		ImplementationReview: task.ReviewConfiguration{Agent: task.AgentReviewConfiguration{Required: true}},
	})
	if err == nil {
		t.Fatal("Specified() error = nil, want an error for a review gate without verification")
	}
}

func TestSpecifiedRejectsMissingPlan(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(
		t.Context(),
		id,
		task.TaskDefinition{Title: "t", Description: "d"},
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := Specified(t.Context(), st, Request{ID: id}); err == nil {
		t.Fatal("Specified() error = nil, want an error for a missing plan")
	}
}
