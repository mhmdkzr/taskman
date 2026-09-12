package md

import (
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
)

func mustApply(t *testing.T, current task.Task, event task.TaskEvent) task.Task {
	t.Helper()
	next, err := task.Apply(current, event)
	if err != nil {
		t.Fatalf("task.Apply(%T) error = %v", event, err)
	}
	return next
}

func TestRenderTaskJustCreated(t *testing.T) {
	now := time.Now().UTC()
	tsk, err := task.NewTask(
		uuid.NewV7(),
		task.TaskDefinition{Title: "Fix drift", Description: "do it", Labels: map[string]string{"priority": "high"}},
		now,
	)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	out, err := RenderTask(tsk)
	if err != nil {
		t.Fatalf("RenderTask() error = %v", err)
	}

	for _, want := range []string{
		"# Task " + tsk.ID.String(),
		"**State:** specify",
		"**Next:** dispatch (specify)",
		"**Title:** Fix drift",
		"**Description:** do it",
		"- priority: high",
		"specify",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
	for _, notWant := range []string{"## Specification", "## Implementation", "## Blocked", "## Abandoned"} {
		if strings.Contains(out, notWant) {
			t.Errorf("output should not contain %q at specify state\n---\n%s", notWant, out)
		}
	}
}

func TestRenderTaskUntitled(t *testing.T) {
	now := time.Now().UTC()
	tsk, err := task.NewTask(uuid.NewV7(), task.TaskDefinition{Description: "do it"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	out, err := RenderTask(tsk)
	if err != nil {
		t.Fatalf("RenderTask() error = %v", err)
	}
	if !strings.Contains(out, "_(none)_") {
		t.Errorf("output should show a placeholder for a missing title\n---\n%s", out)
	}
}

func TestRenderTaskFullLifecycleToCompleted(t *testing.T) {
	now := time.Now().UTC()
	tsk, err := task.NewTask(uuid.NewV7(), task.TaskDefinition{Title: "Ship it", Description: "do it"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	review := task.ReviewConfiguration{
		Agent: task.AgentReviewConfiguration{Required: true, AutoFix: task.AutoFix{Enabled: true, MaxRounds: 2}},
		Human: task.HumanReviewConfiguration{Required: true},
	}
	tsk = mustApply(
		t,
		tsk,
		task.SpecificationSubmitted{Specification: task.Specification{Plan: "the plan", Review: review}, At: now},
	)
	tsk = mustApply(
		t,
		tsk,
		task.SpecificationReviewAgentRejected{
			Findings: []task.Finding{{Location: "spec", Detail: "too vague"}},
			At:       now,
		},
	)
	tsk = mustApply(
		t,
		tsk,
		task.SpecificationSubmitted{
			Specification: task.Specification{Plan: "the better plan", Review: review},
			At:            now,
		},
	)
	tsk = mustApply(t, tsk, task.SpecificationReviewAgentApproved{Comment: "clear now", At: now})
	tsk = mustApply(t, tsk, task.SpecificationReviewHumanApproved{Comment: "lgtm", At: now})

	tsk = mustApply(t, tsk, task.ImplementationCompleted{
		Implementation: task.Implementation{
			Git:          task.Git{Worktree: "/wt", Branch: "task/x"},
			Verification: task.Verification{Tests: task.TestConfiguration{Unit: true}, Linters: true},
			Review:       review,
		},
		At: now,
	})
	tsk = mustApply(
		t,
		tsk,
		task.VerificationFailed{
			Checks: task.Checks{Unit: task.CheckOK, Linters: task.CheckError},
			Output: "lint error",
			At:     now,
		},
	)
	tsk = mustApply(
		t,
		tsk,
		task.VerificationPassed{Checks: task.Checks{Unit: task.CheckOK, Linters: task.CheckOK}, At: now},
	)
	tsk = mustApply(t, tsk, task.ImplementationReviewAgentApproved{Comment: "looks good", At: now})
	tsk = mustApply(
		t,
		tsk,
		task.CommitRecorded{Commit: task.GitCommit{Hash: "c1", Message: "feat: ship it", At: now}, At: now},
	)
	tsk = mustApply(t, tsk, task.ImplementationReviewHumanApproved{Comment: "lgtm", At: now})
	tsk = mustApply(t, tsk, task.MergeCompleted{Merge: task.GitMerge{Target: "main", Commit: "m1", At: now}, At: now})

	out, err := RenderTask(tsk)
	if err != nil {
		t.Fatalf("RenderTask() error = %v", err)
	}

	for _, want := range []string{
		"**State:** completed",
		"**Next:** done (completed)",
		"**Plan:** the better plan",
		// "too vague" (the first submission's rejection) is intentionally
		// gone: resubmitting a specification replaces it wholesale, so
		// review history from before a resubmission never survives.
		"clear now",
		"lgtm",
		"/wt",
		"task/x",
		"`c1` feat: ship it",
		"into `main` at `m1`",
		"unit tests required",
		"linters required",
		"lint error",
		"Auto-fix enabled, max 2 round(s).",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderTaskBlocked(t *testing.T) {
	now := time.Now().UTC()
	tsk, err := task.NewTask(uuid.NewV7(), task.TaskDefinition{Description: "do it"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	tsk = mustApply(t, tsk, task.Escalated{Stage: "definition", Reason: "unclear scope", At: now})

	out, err := RenderTask(tsk)
	if err != nil {
		t.Fatalf("RenderTask() error = %v", err)
	}
	for _, want := range []string{"## Blocked", "**Stage:** definition", "**Reason:** unclear scope"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderTaskAbandoned(t *testing.T) {
	now := time.Now().UTC()
	tsk, err := task.NewTask(uuid.NewV7(), task.TaskDefinition{Description: "do it"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	tsk = mustApply(t, tsk, task.Abandoned{Reason: "no longer needed", At: now})

	out, err := RenderTask(tsk)
	if err != nil {
		t.Fatalf("RenderTask() error = %v", err)
	}
	for _, want := range []string{"## Abandoned", "**Reason:** no longer needed"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderTaskNoReviewOrVerificationRequired(t *testing.T) {
	now := time.Now().UTC()
	tsk, err := task.NewTask(uuid.NewV7(), task.TaskDefinition{Description: "do it"}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	tsk = mustApply(t, tsk, task.SpecificationSubmitted{Specification: task.Specification{Plan: "p"}, At: now})
	tsk = mustApply(
		t,
		tsk,
		task.ImplementationCompleted{
			Implementation: task.Implementation{Git: task.Git{Worktree: "/wt", Branch: "b"}},
			At:             now,
		},
	)

	out, err := RenderTask(tsk)
	if err != nil {
		t.Fatalf("RenderTask() error = %v", err)
	}
	if !strings.Contains(out, "- not required") {
		t.Errorf("output should say verification is not required\n---\n%s", out)
	}
	// Both of specification's gates, and both of implementation's.
	if strings.Count(out, "Not required.") != 4 {
		t.Errorf("output should say all four review gates are not required\n---\n%s", out)
	}
}
