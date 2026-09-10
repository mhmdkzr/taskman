package next

import (
	"strings"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func writeNextTask(t *testing.T, value task.Task) string {
	t.Helper()
	dir := t.TempDir()
	if err := store.Write(dir, value); err != nil {
		t.Fatalf("write task: %v", err)
	}
	return dir
}

func nextTask(state task.State) task.Task {
	return task.Task{
		ID: "abc", State: state, Title: "Test", Definition: "definition",
		Specification: "specification", DoneWhen: "done", Git: task.Git{Worktree: "/tmp/worktree", Branch: "task/test"},
	}
}

func TestNextRendersWorkflowInstructions(t *testing.T) {
	now := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		task       task.Task
		action     Action
		reportWith string
		contains   string
	}{
		{"specify", nextTask(task.StateSpecify), ActionDispatch, "specified abc", "definition"},
		{"specification review", nextTask(task.StateSpecificationReview), ActionWait, "", "awaiting human approval"},
		{"implement", nextTask(task.StateImplement), ActionDispatch, "implemented abc", "specification"},
		{"verify", nextTask(task.StateVerify), ActionRun, "verified abc", "Run the build checks"},
		{"fix failed verification", func() task.Task {
			v := nextTask(task.StateFixVerificationFailure)
			v.Verifications = []task.Verification{
				{Checks: map[string]task.CheckResult{"tests": task.CheckError}, Output: "failed", CreatedAt: now},
			}
			return v
		}(), ActionDispatch, "verified abc", "Verification failed (tests)"},
		{"fix automated review findings", func() task.Task {
			v := nextTask(task.StateFixAutomatedReviewFindings)
			v.Reviews = []task.Review{
				{Attempt: 1, Findings: []task.Finding{{File: "limiter.go", Detail: "shared state"}}, CreatedAt: now},
			}
			return v
		}(), ActionDispatch, "verified abc", "limiter.go: shared state"},
		{
			"automated review",
			nextTask(task.StateAutomatedReview),
			ActionDispatch,
			"reviewed abc",
			"Review the diff",
		},
		{"commit", nextTask(task.StateCommit), ActionDispatch, "committed abc", "Draft a commit message"},
		{"human review", func() task.Task {
			v := nextTask(task.StateHumanReview)
			v.Git.Commit = &task.GitCommit{Hash: "abc123", At: now}
			return v
		}(), ActionWait, "", "abc123"},
		{"merge", nextTask(task.StateMerge), ActionRun, "merged abc", "Merge"},
		{"blocked", func() task.Task {
			v := nextTask(task.StateBlocked)
			v.Blocked = &task.Blocked{
				ResumeState: task.StateImplement,
				Stage:       task.StageImplementation,
				Reason:      "stuck",
				At:          now,
			}
			return v
		}(), ActionWait, "", "stuck"},
		{"completed", nextTask(task.StateCompleted), ActionDone, "", "complete"},
		{
			"abandoned",
			func() task.Task { v := nextTask(task.StateAbandoned); v.FailureReason = "stopped"; return v }(),
			ActionDone,
			"",
			"stopped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeNextTask(t, tt.task)
			got, err := Next(dir, Request{ID: tt.task.ID})
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if got.Action != tt.action {
				t.Errorf("action = %q, want %q", got.Action, tt.action)
			}
			if got.ReportWith != tt.reportWith {
				t.Errorf("report_with = %q, want %q", got.ReportWith, tt.reportWith)
			}
			if !strings.Contains(got.Message, tt.contains) {
				t.Errorf("message does not contain %q:\n%s", tt.contains, got.Message)
			}
		})
	}
}

func TestNextMissingTask(t *testing.T) {
	if _, err := Next(t.TempDir(), Request{ID: "missing"}); err == nil {
		t.Fatal("missing task: want error")
	}
}

func TestNextSpecificationPromptIncludesRejectionFeedback(t *testing.T) {
	value := nextTask(task.StateSpecify)
	value.SpecificationReviews = []task.SpecificationReview{{Approved: false, Comment: "Define the failure behavior"}}
	dir := writeNextTask(t, value)

	got, err := Next(dir, Request{ID: value.ID})
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if !strings.Contains(got.Message, "Define the failure behavior") {
		t.Fatalf("message does not include rejection feedback:\n%s", got.Message)
	}
}
