package task

import (
	"testing"
	"time"
)

// implementedToVerify returns a task in StateVerify whose implementation uses
// impl as its configuration.
func implementedToVerify(t *testing.T, impl Implementation) Task {
	t.Helper()
	now := time.Now().UTC()
	tsk := newTask(t, now)
	tsk = apply(t, tsk, SpecificationSubmitted{Specification: Specification{Plan: "p"}, At: now})
	tsk = apply(t, tsk, ImplementationCompleted{Implementation: impl, At: now})
	requireState(t, tsk, StateVerify)
	return tsk
}

func TestAutoFixStopsVerificationAfterMaxRounds(t *testing.T) {
	now := time.Now().UTC()
	tsk := implementedToVerify(t, Implementation{
		Verification: Verification{
			Tests:   TestConfiguration{Unit: true},
			AutoFix: AutoFix{Enabled: true, MaxRounds: 1},
		},
	})

	tsk = apply(t, tsk, VerificationFailed{Checks: Checks{Unit: CheckError}, At: now})
	requireState(t, tsk, StateFixVerificationFailure)

	tsk = apply(t, tsk, VerificationFailed{Checks: Checks{Unit: CheckError}, At: now})
	requireState(t, tsk, StateBlocked)
	if tsk.Blocked == nil || tsk.Blocked.Stage != "verification" {
		t.Fatalf("Blocked = %+v, want a verification blockage", tsk.Blocked)
	}
}

func TestAutoFixWithUnlimitedRoundsNeverBlocks(t *testing.T) {
	now := time.Now().UTC()
	tsk := implementedToVerify(t, Implementation{
		Verification: Verification{
			Tests:   TestConfiguration{Unit: true},
			AutoFix: AutoFix{Enabled: true},
		},
	})

	for range 5 {
		tsk = apply(t, tsk, VerificationFailed{Checks: Checks{Unit: CheckError}, At: now})
		requireState(t, tsk, StateFixVerificationFailure)
	}
}

func TestAutoFixNotEnabledNeverBlocks(t *testing.T) {
	now := time.Now().UTC()
	tsk := implementedToVerify(t, Implementation{
		Verification: Verification{
			Tests:   TestConfiguration{Unit: true},
			AutoFix: AutoFix{MaxRounds: 1},
		},
	})

	tsk = apply(t, tsk, VerificationFailed{Checks: Checks{Unit: CheckError}, At: now})
	tsk = apply(t, tsk, VerificationFailed{Checks: Checks{Unit: CheckError}, At: now})
	requireState(t, tsk, StateFixVerificationFailure)
}

func TestAutoFixStopsAutomatedReviewAfterMaxRounds(t *testing.T) {
	now := time.Now().UTC()
	tsk := implementedToVerify(t, Implementation{
		Git:          Git{Worktree: "/wt", Branch: "b"},
		Verification: Verification{Tests: TestConfiguration{Unit: true}},
		Review: ReviewConfiguration{Agent: AgentReviewConfiguration{
			Required: true,
			AutoFix:  AutoFix{Enabled: true, MaxRounds: 1},
		}},
	})

	tsk = apply(t, tsk, VerificationPassed{Checks: Checks{Unit: CheckOK}, At: now})
	requireState(t, tsk, StateAutomatedReview)

	tsk = apply(t, tsk, ImplementationReviewAgentRejected{Findings: []Finding{{Location: "f", Detail: "d"}}, At: now})
	requireState(t, tsk, StateFixAutomatedReviewFindings)

	tsk = apply(t, tsk, VerificationPassed{Checks: Checks{Unit: CheckOK}, At: now})
	requireState(t, tsk, StateAutomatedReview)

	tsk = apply(t, tsk, ImplementationReviewAgentRejected{Findings: []Finding{{Location: "f", Detail: "d"}}, At: now})
	requireState(t, tsk, StateBlocked)
	if tsk.Blocked == nil || tsk.Blocked.Stage != "automated review" {
		t.Fatalf("Blocked = %+v, want an automated review blockage", tsk.Blocked)
	}
}

func TestAutoFixStopsHumanReviewAfterMaxRounds(t *testing.T) {
	now := time.Now().UTC()
	tsk := implementedToVerify(t, Implementation{
		Git:          Git{Worktree: "/wt", Branch: "b"},
		Verification: Verification{Tests: TestConfiguration{Unit: true}},
		Review: ReviewConfiguration{Human: HumanReviewConfiguration{
			Required: true,
			AutoFix:  AutoFix{Enabled: true, MaxRounds: 1},
		}},
	})

	tsk = apply(t, tsk, VerificationPassed{Checks: Checks{Unit: CheckOK}, At: now})
	requireState(t, tsk, StateCommit)
	tsk = apply(t, tsk, CommitRecorded{Commit: GitCommit{Hash: "c1", At: now}, At: now})
	requireState(t, tsk, StateHumanReview)

	tsk = apply(t, tsk, ImplementationReviewHumanRejected{Reason: "nope", At: now})
	requireState(t, tsk, StateFixHumanReviewFindings)

	tsk = apply(t, tsk, VerificationPassed{Checks: Checks{Unit: CheckOK}, At: now})
	requireState(t, tsk, StateCommit)
	tsk = apply(t, tsk, CommitRecorded{Commit: GitCommit{Hash: "c2", At: now}, At: now})
	requireState(t, tsk, StateHumanReview)

	tsk = apply(t, tsk, ImplementationReviewHumanRejected{Reason: "nope", At: now})
	requireState(t, tsk, StateBlocked)
	if tsk.Blocked == nil || tsk.Blocked.Stage != "human review" {
		t.Fatalf("Blocked = %+v, want a human review blockage", tsk.Blocked)
	}
}
