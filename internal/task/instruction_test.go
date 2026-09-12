package task

import "testing"

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
