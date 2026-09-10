package task

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func workflowTask(state State) Task {
	return Task{ID: "abc", State: state, Definition: "definition", Specification: "spec", DoneWhen: "done"}
}

func TestApplyTransitions(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	passing := Verification{Checks: map[string]CheckResult{"tests": CheckOK}, CreatedAt: now}
	failing := Verification{Checks: map[string]CheckResult{"tests": CheckError}, CreatedAt: now}
	commit := GitCommit{Hash: "abc123", Message: "feat: work", At: now}

	tests := []struct {
		name  string
		task  Task
		event Event
		want  State
	}{
		{"specification", workflowTask(StateSpecify), SpecificationSubmitted{"spec", "done"}, StateImplement},
		{"implementation", workflowTask(StateImplement), ImplementationCompleted{}, StateVerify},
		{"verification passes", workflowTask(StateVerify), VerificationReported{passing}, StateAutomatedReview},
		{"verification fails", workflowTask(StateVerify), VerificationReported{failing}, StateFixVerificationFailure},
		{
			"verification fix passes",
			workflowTask(StateFixVerificationFailure),
			VerificationReported{passing},
			StateAutomatedReview,
		},
		{
			"verification fix fails",
			workflowTask(StateFixVerificationFailure),
			VerificationReported{failing},
			StateFixVerificationFailure,
		},
		{
			"automated review fix passes",
			workflowTask(StateFixAutomatedReviewFindings),
			VerificationReported{passing},
			StateAutomatedReview,
		},
		{
			"automated review fix fails verification",
			workflowTask(StateFixAutomatedReviewFindings),
			VerificationReported{failing},
			StateFixVerificationFailure,
		},
		{
			"review approves",
			workflowTask(StateAutomatedReview),
			AutomatedReviewRecorded{Approved: true, At: now},
			StateCommit,
		},
		{
			"first review rejects",
			workflowTask(StateAutomatedReview),
			AutomatedReviewRecorded{At: now},
			StateFixAutomatedReviewFindings,
		},
		{
			"second review rejects",
			func() Task { v := workflowTask(StateAutomatedReview); v.Reviews = []Review{{Attempt: 1}}; return v }(),
			AutomatedReviewRecorded{At: now},
			StateBlocked,
		},
		{"commit awaits human", workflowTask(StateCommit), CommitRecorded{commit}, StateHumanReview},
		{
			"auto commit merges",
			func() Task { v := workflowTask(StateCommit); v.AutoApprove = true; return v }(),
			CommitRecorded{commit},
			StateMerge,
		},
		{
			"auto trunk completes",
			func() Task { v := workflowTask(StateCommit); v.AutoApprove = true; v.Git.Trunk = true; return v }(),
			CommitRecorded{commit},
			StateCompleted,
		},
		{"human approves", workflowTask(StateHumanReview), HumanReviewApproved{At: now}, StateMerge},
		{
			"human approves trunk",
			func() Task { v := workflowTask(StateHumanReview); v.Git.Trunk = true; return v }(),
			HumanReviewApproved{At: now},
			StateCompleted,
		},
		{
			"human rejects",
			workflowTask(StateHumanReview),
			HumanReviewRejected{Reason: "fix", At: now},
			StateFixHumanReviewFindings,
		},
		{"human fix passes", workflowTask(StateFixHumanReviewFindings), VerificationReported{passing}, StateCommit},
		{
			"human fix fails",
			workflowTask(StateFixHumanReviewFindings),
			VerificationReported{failing},
			StateFixHumanReviewFindings,
		},
		{"merge", workflowTask(StateMerge), MergeCompleted{}, StateCompleted},
		{
			"escalate",
			workflowTask(StateImplement),
			Escalated{Stage: StageImplementation, Reason: "question", At: now},
			StateBlocked,
		},
		{"abandon", workflowTask(StateImplement), Abandoned{Reason: "stop"}, StateAbandoned},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Apply(tt.task, tt.event)
			if err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if got.State != tt.want {
				t.Fatalf("state = %q, want %q", got.State, tt.want)
			}
		})
	}
}

func TestApplyDoesNotMutateInput(t *testing.T) {
	t.Parallel()
	original := workflowTask(StateVerify)
	original.Labels = map[string]string{"priority": "high"}
	original.Verifications = []Verification{{Checks: map[string]CheckResult{"old": CheckOK}}}
	want := original.Clone()

	_, err := Apply(
		original,
		VerificationReported{Verification: Verification{Checks: map[string]CheckResult{"new": CheckOK}}},
	)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !reflect.DeepEqual(original, want) {
		t.Fatalf("Apply mutated input:\n got: %#v\nwant: %#v", original, want)
	}
}

func TestApplyRejectsInvalidTransition(t *testing.T) {
	t.Parallel()
	_, err := Apply(workflowTask(StateSpecify), ImplementationCompleted{})
	if _, ok := errors.AsType[*InvalidTransitionError](err); !ok {
		t.Fatalf("error = %v, want InvalidTransitionError", err)
	}
}

func TestWorkflowAcceptsOnlyDeclaredStateEvents(t *testing.T) {
	t.Parallel()
	expected := map[State][]EventKind{
		StateSpecify:                    {EventSpecificationSubmitted},
		StateImplement:                  {EventImplementationCompleted},
		StateVerify:                     {EventVerificationReported},
		StateFixVerificationFailure:     {EventVerificationReported},
		StateFixAutomatedReviewFindings: {EventVerificationReported},
		StateAutomatedReview:            {EventAutomatedReviewRecorded},
		StateCommit:                     {EventCommitRecorded},
		StateHumanReview:                {EventHumanReviewApproved, EventHumanReviewRejected},
		StateFixHumanReviewFindings:     {EventVerificationReported},
		StateMerge:                      {EventMergeCompleted},
		StateBlocked:                    nil,
		StateCompleted:                  nil,
		StateAbandoned:                  nil,
	}
	localEvents := []EventKind{
		EventSpecificationSubmitted, EventImplementationCompleted, EventVerificationReported,
		EventAutomatedReviewRecorded, EventCommitRecorded, EventHumanReviewApproved,
		EventHumanReviewRejected, EventMergeCompleted,
	}

	for state, accepted := range expected {
		for _, event := range localEvents {
			wantAccepted := false
			for _, candidate := range accepted {
				wantAccepted = wantAccepted || candidate == event
			}
			err := Accepts(workflowTask(state), event)
			if (err == nil) != wantAccepted {
				t.Errorf("Accepts(%s, %s) error = %v, want accepted=%t", state, event, err, wantAccepted)
			}
		}
	}
}

func TestValidateRejectsInvalidBlockedResumeState(t *testing.T) {
	t.Parallel()
	value := workflowTask(StateBlocked)
	value.Blocked = &Blocked{ResumeState: StateCompleted, Stage: StageImplementation, Reason: "stopped"}
	if err := Validate(value); err == nil {
		t.Fatal("Validate: want invalid resume state error, got nil")
	}
}

func TestNextUsesStateDefinition(t *testing.T) {
	t.Parallel()
	tests := map[State]Instruction{
		StateSpecify:     {InstructionSpecify, InstructionDispatch},
		StateVerify:      {InstructionVerify, InstructionRun},
		StateHumanReview: {InstructionHumanReview, InstructionWait},
		StateCompleted:   {InstructionCompleted, InstructionDone},
	}
	for state, want := range tests {
		got, err := Next(workflowTask(state))
		if err != nil {
			t.Fatalf("Next(%s): %v", state, err)
		}
		if got != want {
			t.Errorf("Next(%s) = %#v, want %#v", state, got, want)
		}
	}
}
