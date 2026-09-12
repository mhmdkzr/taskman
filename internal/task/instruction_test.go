package task

import (
	"slices"
	"testing"
)

func TestInstructionReflectsCurrentState(t *testing.T) {
	tests := []struct {
		state  TaskState
		action InstructionAction
	}{
		{StateSpecify, InstructionDispatch},
		{StateSpecificationReview, InstructionWait},
		{StateImplement, InstructionDispatch},
		{StateVerify, InstructionRun},
		{StateFixVerificationFailure, InstructionDispatch},
		{StateFixAutomatedReviewFindings, InstructionDispatch},
		{StateAutomatedReview, InstructionDispatch},
		{StateCommit, InstructionDispatch},
		{StateHumanReview, InstructionWait},
		{StateFixHumanReviewFindings, InstructionDispatch},
		{StateMerge, InstructionRun},
		{StateBlocked, InstructionWait},
		{StateCompleted, InstructionDone},
		{StateAbandoned, InstructionDone},
	}
	for _, tc := range tests {
		t.Run(string(tc.state), func(t *testing.T) {
			tsk := Task{StateHistory: []StateChange{{State: tc.state}}}
			got := tsk.Instruction()
			if got.Action != tc.action {
				t.Fatalf("Instruction().Action = %s, want %s", got.Action, tc.action)
			}
			if got.State != tc.state {
				t.Fatalf("Instruction().State = %s, want %s", got.State, tc.state)
			}
		})
	}
}

func TestInstructionOnUnknownStateIsDone(t *testing.T) {
	tsk := Task{StateHistory: []StateChange{{State: TaskState("made_up")}}}
	if got := tsk.Instruction().Action; got != InstructionDone {
		t.Fatalf("Instruction().Action = %s, want done", got)
	}
}

func TestValidEventsReflectsGuards(t *testing.T) {
	base := Task{
		StateHistory:  []StateChange{{State: StateSpecificationReview}},
		Specification: &Specification{Plan: "plan"},
	}

	agentOnly := base
	spec := *base.Specification
	spec.Review.Agent.Required = true
	agentOnly.Specification = &spec
	if got := agentOnly.ValidEvents(); !slices.Equal(got, []EventKind{
		EventSpecificationReviewAgentApproved,
		EventSpecificationReviewAgentRejected,
	}) {
		t.Fatalf("ValidEvents() = %v, want only agent events", got)
	}

	both := base
	spec = *base.Specification
	spec.Review.Agent.Required = true
	spec.Review.Human.Required = true
	both.Specification = &spec
	want := []EventKind{
		EventSpecificationReviewAgentApproved,
		EventSpecificationReviewAgentRejected,
		EventSpecificationReviewHumanApproved,
		EventSpecificationReviewHumanRejected,
	}
	slices.Sort(want)
	if got := both.ValidEvents(); !slices.Equal(got, want) {
		t.Fatalf("ValidEvents() = %v, want %v", got, want)
	}
}

func TestValidEventsOnTerminalStateIsEmpty(t *testing.T) {
	tsk := Task{StateHistory: []StateChange{{State: StateCompleted}}}
	if got := tsk.ValidEvents(); len(got) != 0 {
		t.Fatalf("ValidEvents() = %v, want empty", got)
	}
}
