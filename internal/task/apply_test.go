package task

import (
	"errors"
	"testing"
	"time"

	"uuid"
)

func newTask(t *testing.T, at time.Time) Task {
	t.Helper()
	tsk, err := NewTask(uuid.NewV7(), TaskDefinition{Description: "do the work"}, at)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	return tsk
}

func apply(t *testing.T, current Task, event TaskEvent) Task {
	t.Helper()
	next, err := Apply(current, event)
	if err != nil {
		t.Fatalf("Apply(%T) error = %v", event, err)
	}
	return next
}

func requireState(t *testing.T, tsk Task, want TaskState) {
	t.Helper()
	if tsk.State() != want {
		t.Fatalf("state = %s, want %s", tsk.State(), want)
	}
}

func TestApplyMinimalWorkflowCompletesWithoutOptionalGates(t *testing.T) {
	now := time.Now().UTC()
	tsk := newTask(t, now)

	tsk = apply(t, tsk, SpecificationSubmitted{Specification: Specification{Plan: "plan"}, At: now})
	requireState(t, tsk, StateImplement)

	tsk = apply(t, tsk, ImplementationCompleted{
		Implementation: Implementation{Git: Git{Worktree: "/wt", Branch: "b"}},
		At:             now,
	})
	requireState(t, tsk, StateCommit)

	tsk = apply(t, tsk, CommitRecorded{Commit: GitCommit{Hash: "abc", Message: "m", At: now}, At: now})
	requireState(t, tsk, StateMerge)

	tsk = apply(t, tsk, MergeCompleted{Merge: GitMerge{Target: "main", Commit: "def", At: now}, At: now})
	requireState(t, tsk, StateCompleted)

	if got := tsk.Instruction(); got.Action != InstructionDone {
		t.Fatalf("Instruction().Action = %s, want done", got.Action)
	}
	if len(tsk.StateHistory) != 5 {
		t.Fatalf("len(StateHistory) = %d, want 5", len(tsk.StateHistory))
	}
}

func TestApplySpecificationReviewFlow(t *testing.T) {
	now := time.Now().UTC()
	tsk := newTask(t, now)
	review := ReviewConfiguration{
		Agent: AgentReviewConfiguration{Required: true},
		Human: HumanReviewConfiguration{Required: true},
	}

	tsk = apply(t, tsk, SpecificationSubmitted{Specification: Specification{Plan: "plan", Review: review}, At: now})
	requireState(t, tsk, StateSpecificationReview)

	tsk = apply(t, tsk, SpecificationReviewAgentRejected{Findings: []Finding{{Location: "x", Detail: "y"}}, At: now})
	requireState(t, tsk, StateSpecify)

	// Resubmission replaces Specification wholesale - prior agent review
	// results don't carry over, matching "replace on resubmission".
	tsk = apply(t, tsk, SpecificationSubmitted{Specification: Specification{Plan: "plan v2", Review: review}, At: now})
	requireState(t, tsk, StateSpecificationReview)
	if len(tsk.Specification.Review.Agent.Results) != 0 {
		t.Fatalf("agent review results = %d, want 0 after resubmission", len(tsk.Specification.Review.Agent.Results))
	}

	// Agent approves; human review still required, so it stays put.
	tsk = apply(t, tsk, SpecificationReviewAgentApproved{Comment: "looks good", At: now})
	requireState(t, tsk, StateSpecificationReview)

	tsk = apply(t, tsk, SpecificationReviewHumanRejected{Reason: "needs more detail", At: now})
	requireState(t, tsk, StateSpecify)

	tsk = apply(t, tsk, SpecificationSubmitted{Specification: Specification{Plan: "plan v3", Review: review}, At: now})
	tsk = apply(t, tsk, SpecificationReviewAgentApproved{Comment: "ok", At: now})
	tsk = apply(t, tsk, SpecificationReviewHumanApproved{Comment: "approved", At: now})
	requireState(t, tsk, StateImplement)
}

func TestApplyImplementationVerificationAndReviewFlow(t *testing.T) {
	now := time.Now().UTC()
	tsk := newTask(t, now)
	tsk = apply(t, tsk, SpecificationSubmitted{Specification: Specification{Plan: "plan"}, At: now})

	implReview := ReviewConfiguration{
		Agent: AgentReviewConfiguration{Required: true},
		Human: HumanReviewConfiguration{Required: true},
	}
	tsk = apply(t, tsk, ImplementationCompleted{
		Implementation: Implementation{
			Git:          Git{Worktree: "/wt", Branch: "b"},
			Verification: Verification{Tests: TestConfiguration{Unit: true}},
			Review:       implReview,
		},
		At: now,
	})
	requireState(t, tsk, StateVerify)

	tsk = apply(t, tsk, VerificationFailed{Checks: Checks{Unit: CheckError}, At: now})
	requireState(t, tsk, StateFixVerificationFailure)

	tsk = apply(t, tsk, VerificationFailed{Checks: Checks{Unit: CheckError}, At: now})
	requireState(t, tsk, StateFixVerificationFailure)

	tsk = apply(t, tsk, VerificationPassed{Checks: Checks{Unit: CheckOK}, At: now})
	requireState(t, tsk, StateAutomatedReview)
	if len(tsk.Implementation.Verification.Attempts) != 3 {
		t.Fatalf("attempts = %d, want 3", len(tsk.Implementation.Verification.Attempts))
	}

	tsk = apply(t, tsk, ImplementationReviewAgentRejected{Findings: []Finding{{Location: "f", Detail: "d"}}, At: now})
	requireState(t, tsk, StateFixAutomatedReviewFindings)

	tsk = apply(t, tsk, VerificationFailed{Checks: Checks{Unit: CheckError}, At: now})
	requireState(t, tsk, StateFixVerificationFailure)

	tsk = apply(t, tsk, VerificationPassed{Checks: Checks{Unit: CheckOK}, At: now})
	requireState(t, tsk, StateAutomatedReview)

	tsk = apply(t, tsk, ImplementationReviewAgentApproved{Comment: "ok", At: now})
	requireState(t, tsk, StateCommit)

	tsk = apply(t, tsk, CommitRecorded{Commit: GitCommit{Hash: "c1", Message: "m1", At: now}, At: now})
	requireState(t, tsk, StateHumanReview)

	tsk = apply(t, tsk, ImplementationReviewHumanRejected{Reason: "needs work", At: now})
	requireState(t, tsk, StateFixHumanReviewFindings)

	tsk = apply(t, tsk, VerificationFailed{Checks: Checks{Unit: CheckError}, At: now})
	requireState(t, tsk, StateFixHumanReviewFindings)

	tsk = apply(t, tsk, VerificationPassed{Checks: Checks{Unit: CheckOK}, At: now})
	requireState(t, tsk, StateCommit)

	tsk = apply(t, tsk, CommitRecorded{Commit: GitCommit{Hash: "c2", Message: "m2", At: now}, At: now})
	requireState(t, tsk, StateHumanReview)
	if len(tsk.Implementation.Git.Commits) != 2 {
		t.Fatalf("commits = %d, want 2", len(tsk.Implementation.Git.Commits))
	}

	tsk = apply(t, tsk, ImplementationReviewHumanApproved{Comment: "lgtm", At: now})
	requireState(t, tsk, StateMerge)

	tsk = apply(t, tsk, MergeCompleted{Merge: GitMerge{Target: "main", Commit: "m", At: now}, At: now})
	requireState(t, tsk, StateCompleted)
}

func TestApplyEscalationAndAbandonment(t *testing.T) {
	now := time.Now().UTC()
	tsk := newTask(t, now)

	tsk = apply(t, tsk, Escalated{Stage: "definition", Reason: "waiting on input", At: now})
	requireState(t, tsk, StateBlocked)
	if tsk.Blocked == nil || tsk.Blocked.Reason != "waiting on input" {
		t.Fatalf("Blocked = %+v, want reason recorded", tsk.Blocked)
	}

	tsk = apply(t, tsk, Abandoned{Reason: "no longer needed", At: now})
	requireState(t, tsk, StateAbandoned)
	if tsk.Blocked != nil {
		t.Fatal("Blocked should be cleared once abandoned")
	}
	if tsk.Abandoned == nil || tsk.Abandoned.Reason != "no longer needed" {
		t.Fatalf("Abandoned = %+v, want reason recorded", tsk.Abandoned)
	}
}

func TestApplyRejectsInvalidTransitions(t *testing.T) {
	now := time.Now().UTC()

	completed := newTask(t, now)
	completed = apply(t, completed, SpecificationSubmitted{Specification: Specification{Plan: "p"}, At: now})
	completed = apply(t, completed, ImplementationCompleted{Implementation: Implementation{}, At: now})
	completed = apply(t, completed, CommitRecorded{Commit: GitCommit{Hash: "c", At: now}, At: now})
	completed = apply(t, completed, MergeCompleted{Merge: GitMerge{Target: "main", Commit: "c", At: now}, At: now})

	abandoned := newTask(t, now)
	abandoned = apply(t, abandoned, Abandoned{Reason: "moot", At: now})

	blocked := newTask(t, now)
	blocked = apply(t, blocked, Escalated{Stage: "definition", Reason: "stuck", At: now})

	humanReviewOnly := newTask(t, now)
	humanReviewOnly = apply(t, humanReviewOnly, SpecificationSubmitted{
		Specification: Specification{Plan: "p", Review: ReviewConfiguration{Human: HumanReviewConfiguration{Required: true}}},
		At:            now,
	})

	agentReviewOnly := newTask(t, now)
	agentReviewOnly = apply(t, agentReviewOnly, SpecificationSubmitted{
		Specification: Specification{Plan: "p", Review: ReviewConfiguration{Agent: AgentReviewConfiguration{Required: true}}},
		At:            now,
	})

	tests := []struct {
		name    string
		current Task
		event   TaskEvent
	}{
		{
			name:    "commit recorded before implementation",
			current: newTask(t, now),
			event:   CommitRecorded{Commit: GitCommit{Hash: "x", At: now}, At: now},
		},
		{
			name:    "merge completed while specifying",
			current: newTask(t, now),
			event:   MergeCompleted{Merge: GitMerge{Target: "main", Commit: "c", At: now}, At: now},
		},
		{
			name:    "specification approved without human review required",
			current: agentReviewOnly,
			event:   SpecificationReviewHumanApproved{Comment: "ok", At: now},
		},
		{
			name:    "agent review approved without agent review required",
			current: humanReviewOnly,
			event:   SpecificationReviewAgentApproved{Comment: "ok", At: now},
		},
		{
			name:    "event reported after completion",
			current: completed,
			event:   Escalated{Stage: "definition", Reason: "too late", At: now},
		},
		{
			name:    "event reported after abandonment",
			current: abandoned,
			event:   SpecificationSubmitted{Specification: Specification{Plan: "p"}, At: now},
		},
		{
			name:    "escalate while already blocked",
			current: blocked,
			event:   Escalated{Stage: "definition", Reason: "again", At: now},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Apply(tc.current, tc.event); !errors.Is(err, errInvalidTransition) {
				t.Fatalf("Apply() error = %v, want errInvalidTransition", err)
			}
		})
	}
}

func TestApplyRejectsInvalidEventPayloads(t *testing.T) {
	now := time.Now().UTC()

	implementing := newTask(t, now)
	implementing = apply(t, implementing, SpecificationSubmitted{Specification: Specification{Plan: "p"}, At: now})
	verifying := apply(t, implementing, ImplementationCompleted{
		Implementation: Implementation{Verification: Verification{Tests: TestConfiguration{Unit: true}}},
		At:             now,
	})
	committing := apply(t, implementing, ImplementationCompleted{Implementation: Implementation{}, At: now})

	humanReviewOnly := newTask(t, now)
	humanReviewOnly = apply(t, humanReviewOnly, SpecificationSubmitted{
		Specification: Specification{Plan: "p", Review: ReviewConfiguration{Human: HumanReviewConfiguration{Required: true}}},
		At:            now,
	})

	agentReviewOnly := newTask(t, now)
	agentReviewOnly = apply(t, agentReviewOnly, SpecificationSubmitted{
		Specification: Specification{Plan: "p", Review: ReviewConfiguration{Agent: AgentReviewConfiguration{Required: true}}},
		At:            now,
	})

	tests := []struct {
		name    string
		current Task
		event   TaskEvent
	}{
		{
			name:    "specification submitted without a plan",
			current: newTask(t, now),
			event:   SpecificationSubmitted{Specification: Specification{}, At: now},
		},
		{
			name:    "specification rejected without a reason",
			current: humanReviewOnly,
			event:   SpecificationReviewHumanRejected{Reason: "", At: now},
		},
		{
			name:    "agent review rejected without findings",
			current: agentReviewOnly,
			event:   SpecificationReviewAgentRejected{Findings: nil, At: now},
		},
		{
			name:    "verification passed without checks",
			current: verifying,
			event:   VerificationPassed{At: now},
		},
		{
			name:    "verification failed without checks",
			current: verifying,
			event:   VerificationFailed{At: now},
		},
		{
			name:    "commit recorded without a hash",
			current: committing,
			event:   CommitRecorded{Commit: GitCommit{Hash: ""}, At: now},
		},
		{
			name:    "escalated without a reason",
			current: newTask(t, now),
			event:   Escalated{Stage: "definition", Reason: "", At: now},
		},
		{
			name:    "escalated without a stage",
			current: newTask(t, now),
			event:   Escalated{Stage: "", Reason: "stuck", At: now},
		},
		{
			name:    "abandoned without a reason",
			current: newTask(t, now),
			event:   Abandoned{Reason: "", At: now},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Apply(tc.current, tc.event); err == nil {
				t.Fatal("Apply() error = nil, want an error")
			}
		})
	}
}

func TestApplyRejectsNilEvent(t *testing.T) {
	now := time.Now().UTC()
	if _, err := Apply(newTask(t, now), nil); err == nil {
		t.Fatal("Apply() error = nil, want an error for nil event")
	}
}

func TestApplyRejectsUnknownCurrentState(t *testing.T) {
	// Validate already rejects an unrecognized state, so this is caught before
	// Apply's own errUnknownState check (which guards against workflow.states
	// drifting out of sync with TaskState.valid() - not reachable here, but
	// worth keeping as a safety net against that drift).
	tsk := Task{
		ID:           uuid.NewV7(),
		Definition:   TaskDefinition{Description: "d"},
		StateHistory: []StateChange{{State: TaskState("made_up")}},
	}
	if _, err := Apply(tsk, Abandoned{Reason: "x"}); err == nil {
		t.Fatal("Apply() error = nil, want an error for an unrecognized state")
	}
}

func TestApplyDoesNotMutateCurrent(t *testing.T) {
	now := time.Now().UTC()
	tsk := newTask(t, now)
	tsk = apply(t, tsk, SpecificationSubmitted{
		Specification: Specification{Plan: "p", Review: ReviewConfiguration{Agent: AgentReviewConfiguration{Required: true}}},
		At:            now,
	})
	before := tsk.Clone()

	if _, err := Apply(tsk, SpecificationReviewAgentApproved{Comment: "ok", At: now}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if tsk.State() != before.State() {
		t.Fatalf("current mutated: state = %s, want %s", tsk.State(), before.State())
	}
	if len(tsk.Specification.Review.Agent.Results) != len(before.Specification.Review.Agent.Results) {
		t.Fatal("current mutated: agent review results changed")
	}
}

func TestApplyOnFailureDoesNotMutateCurrent(t *testing.T) {
	now := time.Now().UTC()
	tsk := newTask(t, now)
	before := tsk.Clone()

	if _, err := Apply(tsk, CommitRecorded{Commit: GitCommit{Hash: "x", At: now}, At: now}); err == nil {
		t.Fatal("Apply() error = nil, want an error")
	}

	if tsk.State() != before.State() || tsk.Specification != nil {
		t.Fatal("current mutated on a failed Apply")
	}
}

// The remaining reducer branches below are unreachable through Apply given
// the compiled workflow (Implementation and Merge are each only ever set
// once, from states that structurally guarantee their preconditions), but
// they guard those write-once invariants independent of the table's current
// shape, so they're exercised directly.

func TestRecordImplementationRejectsWhenAlreadyPresent(t *testing.T) {
	tsk := Task{Implementation: &Implementation{}}
	err := recordImplementation(&tsk, ImplementationCompleted{Implementation: Implementation{}})
	if !errors.Is(err, errAlreadyPresent) {
		t.Fatalf("recordImplementation() error = %v, want errAlreadyPresent", err)
	}
}

func TestRecordMergeRejectsWithoutSourceCommit(t *testing.T) {
	tsk := Task{Implementation: &Implementation{}}
	err := recordMerge(&tsk, MergeCompleted{Merge: GitMerge{Target: "main", Commit: "c"}})
	if !errors.Is(err, errMissingTaskData) {
		t.Fatalf("recordMerge() error = %v, want errMissingTaskData", err)
	}
}

func TestRecordMergeRejectsWhenAlreadyMerged(t *testing.T) {
	tsk := Task{Implementation: &Implementation{Git: Git{
		Commits: []GitCommit{{Hash: "c1"}},
		Merge:   &GitMerge{Target: "main", Commit: "c1"},
	}}}
	err := recordMerge(&tsk, MergeCompleted{Merge: GitMerge{Target: "main", Commit: "c2"}})
	if !errors.Is(err, errAlreadyPresent) {
		t.Fatalf("recordMerge() error = %v, want errAlreadyPresent", err)
	}
}

func TestReducersRejectMismatchedEventPayload(t *testing.T) {
	tests := []struct {
		name       string
		fn         reducer
		wrongEvent TaskEvent
	}{
		{"recordSpecification", recordSpecification, Abandoned{}},
		{"recordSpecificationReviewHumanApproved", recordSpecificationReviewHumanApproved, Abandoned{}},
		{"recordSpecificationReviewHumanRejected", recordSpecificationReviewHumanRejected, Abandoned{}},
		{"recordSpecificationReviewAgentApproved", recordSpecificationReviewAgentApproved, Abandoned{}},
		{"recordSpecificationReviewAgentRejected", recordSpecificationReviewAgentRejected, Abandoned{}},
		{"recordImplementation", recordImplementation, Abandoned{}},
		{"recordVerificationPassed", recordVerificationPassed, Abandoned{}},
		{"recordVerificationFailed", recordVerificationFailed, Abandoned{}},
		{"recordImplementationReviewAgentApproved", recordImplementationReviewAgentApproved, Abandoned{}},
		{"recordImplementationReviewAgentRejected", recordImplementationReviewAgentRejected, Abandoned{}},
		{"recordCommit", recordCommit, Abandoned{}},
		{"recordImplementationReviewHumanApproved", recordImplementationReviewHumanApproved, Abandoned{}},
		{"recordImplementationReviewHumanRejected", recordImplementationReviewHumanRejected, Abandoned{}},
		{"recordMerge", recordMerge, Abandoned{}},
		{"recordEscalation", recordEscalation, Abandoned{}},
		{"recordAbandonment", recordAbandonment, Escalated{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tsk := Task{Specification: &Specification{}, Implementation: &Implementation{}}
			if err := tc.fn(&tsk, tc.wrongEvent); !errors.Is(err, errInvalidEventPayload) {
				t.Fatalf("%s() error = %v, want errInvalidEventPayload", tc.name, err)
			}
		})
	}
}
