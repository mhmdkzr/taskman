package utils

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
)

func TestSplitKV(t *testing.T) {
	got, err := SplitKV([]string{"a=1", "b=two=parts"})
	if err != nil {
		t.Fatalf("SplitKV: %v", err)
	}
	if got["a"] != "1" || got["b"] != "two=parts" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseChecks(t *testing.T) {
	got, err := ParseChecks([]string{"test=ok", "lint=error"})
	if err != nil {
		t.Fatalf("ParseChecks: %v", err)
	}
	if got["test"] != task.CheckOK || got["lint"] != task.CheckError {
		t.Fatalf("got %#v", got)
	}
	if _, err := ParseChecks([]string{"test=maybe"}); err == nil {
		t.Fatal("invalid check: want error")
	}
}

func TestParseFindings(t *testing.T) {
	got, err := ParseFindings([]string{"main.go=bad"})
	if err != nil {
		t.Fatalf("ParseFindings: %v", err)
	}
	if len(got) != 1 || got[0].File != "main.go" || got[0].Detail != "bad" {
		t.Fatalf("got %#v", got)
	}
}

func TestCurrentStage(t *testing.T) {
	tests := []struct {
		name  string
		state task.State
		want  string
	}{
		{"specify", task.StateSpecify, "awaiting specification"},
		{"implement", task.StateImplement, "awaiting implementation"},
		{"verify", task.StateVerify, "awaiting verification"},
		{"commit", task.StateCommit, "awaiting commit"},
		{"human review", task.StateHumanReview, "awaiting human review"},
		{"merge", task.StateMerge, "awaiting merge"},
		{"completed", task.StateCompleted, "completed"},
		{"abandoned", task.StateAbandoned, "abandoned"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CurrentStage(task.Task{State: tt.state}); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFailMapsErrorsToExitCodes(t *testing.T) {
	if got := ExitCode(
		Fail(&task.InvalidTransitionError{State: task.StateVerify, Event: task.EventCommitRecorded}),
	); got != 1 {
		t.Errorf("InvalidTransitionError exit = %d, want 1", got)
	}
}
