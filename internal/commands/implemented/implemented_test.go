package implemented

import (
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/git"
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

func TestImplemented(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Title: "t", Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := st.Append(
		t.Context(),
		id,
		task.SpecificationSubmitted{
			Specification: task.Specification{
				Plan:     "p",
				Worktree: task.WorktreePolicy{UseWorktree: true, Worktree: "/wt", Branch: "b"},
			},
			At: now,
		},
	); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}

	got, err := Implemented(t.Context(), st, git.NewClient(t.TempDir()), Request{ID: id})
	if err != nil {
		t.Fatalf("Implemented() error = %v", err)
	}
	if got.State() != task.StateCommit {
		t.Fatalf("state = %s, want %s", got.State(), task.StateCommit)
	}
	if got.Implementation.Git.Worktree != "/wt" || got.Implementation.Git.Branch != "b" {
		t.Fatalf("git = %+v, want worktree=/wt branch=b", got.Implementation.Git)
	}
}

func TestSpecifiedRejectsImplementationReviewWithoutVerification(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Title: "t", Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err := st.Append(
		t.Context(),
		id,
		task.SpecificationSubmitted{
			Specification: task.Specification{
				Plan:                 "p",
				ImplementationReview: task.ReviewConfiguration{Agent: task.AgentReviewConfiguration{Required: true}},
			},
			At: now,
		},
	)
	if err == nil {
		t.Fatal("Append(SpecificationSubmitted) error = nil, want an error for a review gate without verification")
	}
}

func TestImplementedRejectsUnspecifiedPolicy(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Title: "t", Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	// Simulates a task specified before implementation policy moved to
	// `specified`: a plan with none of the new policy fields set.
	if _, err := st.Append(
		t.Context(), id, task.SpecificationSubmitted{Specification: task.Specification{Plan: "p"}, At: now},
	); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}

	if _, err := Implemented(t.Context(), st, git.NewClient(t.TempDir()), Request{ID: id}); err == nil {
		t.Fatal("Implemented() error = nil, want an error for a specification with no implementation policy")
	}
}
