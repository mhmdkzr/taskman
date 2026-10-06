package task

import (
	"errors"
	"testing"
	"time"
)

// amendable builds a task in specification_review with both specification
// reviews required and the agent gate already approved, so an amendment has a
// recorded review result to preserve or discard.
func amendable(t *testing.T, now time.Time) Task {
	t.Helper()
	tsk := newTask(t, now)
	review := ReviewConfiguration{
		Agent: AgentReviewConfiguration{Required: true},
		Human: HumanReviewConfiguration{Required: true},
	}
	spec := Specification{
		Plan:         "plan",
		Review:       review,
		Verification: Verification{Tests: TestConfiguration{Unit: true}},
	}
	tsk = apply(t, tsk, SpecificationSubmitted{Specification: spec, At: now})
	tsk = apply(t, tsk, SpecificationReviewAgentApproved{Comment: "ok", At: now})
	requireState(t, tsk, StateSpecificationReview)
	return tsk
}

func TestApplyAmendmentPreservesReviewWhenInputsUnchanged(t *testing.T) {
	now := time.Now().UTC()
	tsk := amendable(t, now)
	if len(tsk.Specification.Review.Agent.Results) != 1 {
		t.Fatalf("setup: agent results = %d, want 1", len(tsk.Specification.Review.Agent.Results))
	}

	// Only implementation policy changes: add an integration check.
	amended := *tsk.Specification
	amended.Verification.Tests.Integration = true
	tsk = apply(t, tsk, SpecificationAmended{Specification: amended, At: now})

	requireState(t, tsk, StateSpecificationReview)
	if !tsk.Specification.Verification.Tests.Integration {
		t.Fatal("integration check not applied")
	}
	if len(tsk.Specification.Review.Agent.Results) != 1 {
		t.Fatalf("agent results = %d, want preserved", len(tsk.Specification.Review.Agent.Results))
	}
}

func TestApplyAmendmentClearsReviewWhenPlanChanges(t *testing.T) {
	now := time.Now().UTC()
	tsk := amendable(t, now)

	amended := *tsk.Specification
	amended.Plan = "revised plan"
	tsk = apply(t, tsk, SpecificationAmended{Specification: amended, At: now})

	requireState(t, tsk, StateSpecificationReview)
	if tsk.Specification.Plan != "revised plan" {
		t.Fatalf("plan = %q, want revised", tsk.Specification.Plan)
	}
	if len(tsk.Specification.Review.Agent.Results) != 0 {
		t.Fatalf("agent results = %d, want cleared", len(tsk.Specification.Review.Agent.Results))
	}
}

func TestApplyAmendmentRemovingReviewsAdvancesToImplement(t *testing.T) {
	now := time.Now().UTC()
	tsk := amendable(t, now)

	amended := *tsk.Specification
	amended.Review = ReviewConfiguration{}
	tsk = apply(t, tsk, SpecificationAmended{Specification: amended, At: now})

	requireState(t, tsk, StateImplement)
}

func TestApplyAmendmentFromImplementRerunsReview(t *testing.T) {
	now := time.Now().UTC()
	tsk := newTask(t, now)
	review := ReviewConfiguration{Agent: AgentReviewConfiguration{Required: true}}
	spec := Specification{
		Plan:         "plan",
		Review:       review,
		Verification: Verification{Tests: TestConfiguration{Unit: true}},
	}
	tsk = apply(t, tsk, SpecificationSubmitted{Specification: spec, At: now})
	tsk = apply(t, tsk, SpecificationReviewAgentApproved{Comment: "ok", At: now})
	requireState(t, tsk, StateImplement)

	amended := *tsk.Specification
	amended.Plan = "revised plan"
	tsk = apply(t, tsk, SpecificationAmended{Specification: amended, At: now})

	requireState(t, tsk, StateSpecificationReview)
	if len(tsk.Specification.Review.Agent.Results) != 0 {
		t.Fatalf("agent results = %d, want cleared", len(tsk.Specification.Review.Agent.Results))
	}
}

func TestApplyAmendmentFromImplementKeepsPositionWhenOnlyPolicyChanges(t *testing.T) {
	now := time.Now().UTC()
	tsk := newTask(t, now)
	review := ReviewConfiguration{Agent: AgentReviewConfiguration{Required: true}}
	spec := Specification{
		Plan:         "plan",
		Review:       review,
		Verification: Verification{Tests: TestConfiguration{Unit: true}},
	}
	tsk = apply(t, tsk, SpecificationSubmitted{Specification: spec, At: now})
	tsk = apply(t, tsk, SpecificationReviewAgentApproved{Comment: "ok", At: now})
	requireState(t, tsk, StateImplement)

	amended := *tsk.Specification
	amended.Verification.Tests.Integration = true
	tsk = apply(t, tsk, SpecificationAmended{Specification: amended, At: now})

	requireState(t, tsk, StateImplement)
	if len(tsk.Specification.Review.Agent.Results) != 1 {
		t.Fatalf("agent results = %d, want preserved", len(tsk.Specification.Review.Agent.Results))
	}
}

func TestApplyAmendmentRejectedOutsidePreImplementation(t *testing.T) {
	now := time.Now().UTC()
	tsk := newTask(t, now)
	tsk = apply(t, tsk, SpecificationSubmitted{Specification: Specification{Plan: "plan"}, At: now})
	tsk = apply(t, tsk, ImplementationCompleted{
		Implementation: Implementation{Git: Git{Worktree: "/wt", Branch: "b"}},
		At:             now,
	})
	requireState(t, tsk, StateCommit)

	amended := *tsk.Specification
	amended.Plan = "too late"
	event := SpecificationAmended{Specification: amended, At: now}
	if _, err := Apply(tsk, event); !errors.Is(err, errInvalidTransition) {
		t.Fatalf("Apply() error = %v, want errInvalidTransition", err)
	}
}

func TestApplyAmendmentRejectsMissingPlan(t *testing.T) {
	now := time.Now().UTC()
	tsk := amendable(t, now)

	amended := *tsk.Specification
	amended.Plan = ""
	if _, err := Apply(tsk, SpecificationAmended{Specification: amended, At: now}); err == nil {
		t.Fatal("Apply() error = nil, want an error for a missing plan")
	}
}

func TestSpecReviewInputsChanged(t *testing.T) {
	base := Specification{
		Plan:   "p",
		Review: ReviewConfiguration{Agent: AgentReviewConfiguration{Required: true}},
	}
	tests := []struct {
		name string
		mut  func(*Specification)
		want bool
	}{
		{"identical", func(*Specification) {}, false},
		{"impl policy only", func(s *Specification) { s.Verification.Linters = true }, false},
		{"plan", func(s *Specification) { s.Plan = "q" }, true},
		{"agent required", func(s *Specification) { s.Review.Agent.Required = false }, true},
		{"agent autofix", func(s *Specification) { s.Review.Agent.AutoFix.Enabled = true }, true},
		{"human required", func(s *Specification) { s.Review.Human.Required = true }, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			amended := base
			tc.mut(&amended)
			if got := specReviewInputsChanged(base, amended); got != tc.want {
				t.Fatalf("specReviewInputsChanged() = %v, want %v", got, tc.want)
			}
		})
	}
}
