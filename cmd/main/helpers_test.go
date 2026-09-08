package main

import (
	"testing"

	"github.com/mhmdkzr/loop/internal/task"
)

func TestSplitKV(t *testing.T) {
	got, err := splitKV([]string{"priority=high", "note=has=equals=too"})
	if err != nil {
		t.Fatalf("splitKV: %v", err)
	}
	if got["priority"] != "high" || got["note"] != "has=equals=too" {
		t.Errorf("splitKV = %+v", got)
	}

	if _, err := splitKV([]string{"malformed"}); err == nil {
		t.Error("splitKV(malformed): want error, got nil")
	}
}

func TestParseChecks(t *testing.T) {
	got, err := parseChecks([]string{"vet=ok", "lint=error"})
	if err != nil {
		t.Fatalf("parseChecks: %v", err)
	}
	if got["vet"] != task.CheckOK || got["lint"] != task.CheckError {
		t.Errorf("parseChecks = %+v", got)
	}

	if _, err := parseChecks([]string{"vet=maybe"}); err == nil {
		t.Error("parseChecks(vet=maybe): want error, got nil")
	}
}

func TestParseFindings(t *testing.T) {
	got, err := parseFindings([]string{"a.go=missing check", "b.go=unused var"})
	if err != nil {
		t.Fatalf("parseFindings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("parseFindings len = %d, want 2", len(got))
	}
	byFile := map[string]string{}
	for _, f := range got {
		byFile[f.File] = f.Detail
	}
	if byFile["a.go"] != "missing check" || byFile["b.go"] != "unused var" {
		t.Errorf("parseFindings = %+v", byFile)
	}
}

func TestCurrentStage(t *testing.T) {
	cases := []struct {
		name string
		t    task.Task
		want string
	}{
		{
			"awaiting spec",
			task.Task{
				State:  task.StateCreated,
				Status: task.Status{Specification: task.StageStatus{State: task.StagePending}},
			},
			"awaiting specification",
		},
		{"awaiting impl", task.Task{State: task.StateStarted, Status: task.Status{
			Specification: task.StageStatus{
				State: task.StageDone,
			},
			Implementation: task.StageStatus{State: task.StagePending},
		}}, "awaiting implementation"},
		{"in verification", task.Task{State: task.StateStarted, Status: task.Status{
			Specification: task.StageStatus{
				State: task.StageDone,
			},
			Implementation: task.StageStatus{State: task.StageDone},
			Verification:   task.StageStatus{State: task.StagePending, Attempts: 1},
		}}, "in verification (attempt 1)"},
		{
			"blocked",
			task.Task{State: task.StateBlocked, Blocked: &task.Blocked{Stage: "verification"}},
			"blocked in verification",
		},
		{"completed", task.Task{State: task.StateCompleted}, "merged"},
		{"failed", task.Task{State: task.StateFailed}, "abandoned"},
	}
	for _, c := range cases {
		if got := currentStage(c.t); got != c.want {
			t.Errorf("%s: currentStage() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFailMapsErrorsToExitCodes(t *testing.T) {
	if got := exitCode(fail(task.ErrTaskNotFound)); got != 1 {
		t.Errorf("ErrTaskNotFound exit = %d, want 1", got)
	}
	if got := exitCode(fail(task.ErrInvalidLabel)); got != 2 {
		t.Errorf("ErrInvalidLabel exit = %d, want 2", got)
	}
	if got := exitCode(fail(&task.InvalidTransitionError{Stage: "review", Have: "pending", Want: "done"})); got != 1 {
		t.Errorf("InvalidTransitionError exit = %d, want 1", got)
	}
}
