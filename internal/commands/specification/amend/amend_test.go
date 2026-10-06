package amend

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

func specified(t *testing.T, st *store.Store, spec task.Specification) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	definition := task.TaskDefinition{Title: "t", Description: "d"}
	if _, err := st.Create(t.Context(), id, definition, time.Now().UTC()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	event := task.SpecificationSubmitted{Specification: spec, At: time.Now().UTC()}
	if _, err := st.Append(t.Context(), id, event); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}
	return id
}

func TestAmendPatchesOnlySuppliedFields(t *testing.T) {
	st := openTestStore(t)
	id := specified(t, st, task.Specification{
		Plan:                 "original plan",
		Review:               task.ReviewConfiguration{Agent: task.AgentReviewConfiguration{Required: true}},
		Verification:         task.Verification{Tests: task.TestConfiguration{Unit: true}},
		ImplementationReview: task.ReviewConfiguration{Human: task.HumanReviewConfiguration{Required: true}},
	})

	got, err := Amend(t.Context(), st, Request{
		ID:           id,
		Verification: &VerificationPatch{Tests: &TestPatch{Integration: new(true)}},
	})
	if err != nil {
		t.Fatalf("Amend() error = %v", err)
	}
	spec := got.Specification
	if spec.Plan != "original plan" {
		t.Fatalf("plan = %q, want unchanged", spec.Plan)
	}
	if !spec.Review.Agent.Required {
		t.Fatal("agent review requirement was dropped by an unrelated patch")
	}
	if !spec.Verification.Tests.Unit || !spec.Verification.Tests.Integration {
		t.Fatalf("tests = %+v, want unit and integration", spec.Verification.Tests)
	}
	if !spec.ImplementationReview.Human.Required {
		t.Fatal("implementation review requirement was dropped by an unrelated patch")
	}
}

func TestAmendPlanChangeRerunsReview(t *testing.T) {
	st := openTestStore(t)
	id := specified(t, st, task.Specification{
		Plan:         "original plan",
		Review:       task.ReviewConfiguration{Agent: task.AgentReviewConfiguration{Required: true}},
		Verification: task.Verification{Tests: task.TestConfiguration{Unit: true}},
	})
	approved := task.SpecificationReviewAgentApproved{Comment: "ok", At: time.Now().UTC()}
	if _, err := st.Append(t.Context(), id, approved); err != nil {
		t.Fatalf("Append(agent approved) error = %v", err)
	}

	got, err := Amend(t.Context(), st, Request{ID: id, Plan: new("revised plan")})
	if err != nil {
		t.Fatalf("Amend() error = %v", err)
	}
	if got.State() != task.StateSpecificationReview {
		t.Fatalf("state = %s, want %s", got.State(), task.StateSpecificationReview)
	}
	if len(got.Specification.Review.Agent.Results) != 0 {
		t.Fatalf("agent results = %d, want cleared", len(got.Specification.Review.Agent.Results))
	}
}

func TestAmendPolicyOnlyKeepsReviewResult(t *testing.T) {
	st := openTestStore(t)
	id := specified(t, st, task.Specification{
		Plan: "original plan",
		Review: task.ReviewConfiguration{
			Agent: task.AgentReviewConfiguration{Required: true},
			Human: task.HumanReviewConfiguration{Required: true},
		},
		Verification: task.Verification{Tests: task.TestConfiguration{Unit: true}},
	})
	approved := task.SpecificationReviewAgentApproved{Comment: "ok", At: time.Now().UTC()}
	if _, err := st.Append(t.Context(), id, approved); err != nil {
		t.Fatalf("Append(agent approved) error = %v", err)
	}

	got, err := Amend(t.Context(), st, Request{
		ID:           id,
		Verification: &VerificationPatch{Linters: new(true)},
	})
	if err != nil {
		t.Fatalf("Amend() error = %v", err)
	}
	if got.State() != task.StateSpecificationReview {
		t.Fatalf("state = %s, want %s", got.State(), task.StateSpecificationReview)
	}
	if len(got.Specification.Review.Agent.Results) != 1 {
		t.Fatalf("agent results = %d, want preserved", len(got.Specification.Review.Agent.Results))
	}
}

func TestAmendRejectsEmptyPatch(t *testing.T) {
	st := openTestStore(t)
	id := specified(t, st, task.Specification{Plan: "p"})
	if _, err := Amend(t.Context(), st, Request{ID: id}); err == nil {
		t.Fatal("Amend() error = nil, want an error for an empty patch")
	}
}

func TestAmendRejectsTaskWithoutSpecification(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	definition := task.TaskDefinition{Title: "t", Description: "d"}
	if _, err := st.Create(t.Context(), id, definition, time.Now().UTC()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := Amend(t.Context(), st, Request{ID: id, Plan: new("p")}); err == nil {
		t.Fatal("Amend() error = nil, want an error for a task with no specification")
	}
}

func TestAmendRejectsAfterImplementationRecorded(t *testing.T) {
	st := openTestStore(t)
	id := specified(t, st, task.Specification{Plan: "p"})
	event := task.ImplementationCompleted{
		Implementation: task.Implementation{Git: task.Git{Worktree: "/wt", Branch: "b"}},
		At:             time.Now().UTC(),
	}
	if _, err := st.Append(t.Context(), id, event); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}
	if _, err := Amend(t.Context(), st, Request{ID: id, Plan: new("too late")}); err == nil {
		t.Fatal("Amend() error = nil, want an error after implementation is recorded")
	}
}
