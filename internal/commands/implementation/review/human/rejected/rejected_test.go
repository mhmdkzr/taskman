package rejected

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

func TestRejected(t *testing.T) {
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
		Implementation: task.Implementation{
			Git:          task.Git{Worktree: "/wt", Branch: "b"},
			Verification: task.Verification{Tests: task.TestConfiguration{Unit: true}},
			Review: task.ReviewConfiguration{
				Human: task.HumanReviewConfiguration{Required: true},
			},
		},
		At: now,
	}); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}
	if _, err := st.Append(
		t.Context(),
		id,
		task.VerificationPassed{Checks: task.Checks{Unit: task.CheckOK}, At: now},
	); err != nil {
		t.Fatalf("Append(VerificationPassed) error = %v", err)
	}
	if _, err := st.Append(
		t.Context(),
		id,
		task.CommitRecorded{Commit: task.GitCommit{Hash: "c", At: now}, At: now},
	); err != nil {
		t.Fatalf("Append(CommitRecorded) error = %v", err)
	}

	got, err := Rejected(t.Context(), st, Request{ID: id, Reason: "needs work"})
	if err != nil {
		t.Fatalf("Rejected() error = %v", err)
	}
	if got.State() != task.StateFixHumanReviewFindings {
		t.Fatalf("state = %s, want %s", got.State(), task.StateFixHumanReviewFindings)
	}
}

func TestRejectedRejectsMissingReason(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := Rejected(t.Context(), st, Request{ID: id}); err == nil {
		t.Fatal("Rejected() error = nil, want an error for a missing reason")
	}
}
