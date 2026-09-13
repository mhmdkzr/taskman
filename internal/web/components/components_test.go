package components

import (
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
)

func TestTaskListRendersFullDetail(t *testing.T) {
	at := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)
	tk := task.Task{
		ID: uuid.NewV7(),
		Definition: task.TaskDefinition{
			Title:       "Ship the feature",
			Description: "# Heading\n\nSome **bold** text.",
			Labels:      map[string]string{"area": "ui"},
		},
		StateHistory: []task.StateChange{
			{State: task.StateSpecify, At: at},
			{State: task.StateImplement, At: at.Add(time.Hour)},
			{State: task.StateCompleted, At: at.Add(2 * time.Hour)},
		},
		Specification: &task.Specification{
			Plan: "1. do it",
			Review: task.ReviewConfiguration{
				Agent: task.AgentReviewConfiguration{
					Required: true,
					Results: []task.AgentReviewResult{{
						Comment:  "needs work",
						Findings: []task.Finding{{Location: "spec.md:1", Detail: "too vague"}},
						At:       at,
					}},
				},
				Human: task.HumanReviewConfiguration{
					Required: true,
					Results:  []task.HumanReviewResult{{Approved: true, Comment: "lgtm", At: at}},
				},
			},
		},
		Implementation: &task.Implementation{
			Git: task.Git{
				Worktree: "/tmp/wt",
				Branch:   "feat/x",
				Commits:  []task.GitCommit{{Hash: "abc123", Message: "feat: x", At: at}},
				Merge:    &task.GitMerge{Target: "main", Commit: "def456", At: at},
			},
			Verification: task.Verification{
				Tests:   task.TestConfiguration{Unit: true},
				Linters: true,
				Attempts: []task.VerificationResult{{
					Passed: true,
					Checks: task.Checks{Unit: task.CheckOK, Linters: task.CheckError},
					Output: "ok",
					At:     at,
				}},
			},
			Review: task.ReviewConfiguration{
				Agent: task.AgentReviewConfiguration{
					Required: true,
					Results:  []task.AgentReviewResult{{Approved: true, At: at}},
				},
			},
		},
	}

	var b strings.Builder
	if err := TaskList([]task.Task{tk}).Render(t.Context(), &b); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	html := b.String()

	wants := []string{
		"Ship the feature",
		"Completed",
		"<h1", // markdown description heading
		"Specification plan",
		"feat/x",            // branch
		"/tmp/wt",           // worktree
		"abc123",            // latest commit hash
		"feat: x",           // latest commit message
		"main",              // merge target
		"def456",            // merge commit
		"Agent · attempt 1", // specification agent review
		"needs work",        // specification agent review comment
		"spec.md:1",         // specification finding location
		"lgtm",              // specification human review comment
		"Verification",
		"Attempt 1", // verification attempt label
		"State history",
	}
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Errorf("rendered output missing %q", want)
		}
	}
}

func TestTaskListRendersBareTask(t *testing.T) {
	tk := task.Task{
		ID:           uuid.NewV7(),
		Definition:   task.TaskDefinition{Description: "just created"},
		StateHistory: []task.StateChange{{State: task.StateSpecify}},
	}

	var b strings.Builder
	if err := TaskList([]task.Task{tk}).Render(t.Context(), &b); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(b.String(), "just created") {
		t.Errorf("rendered output missing description")
	}
}

func TestGroupTasksOrdersAttentionFirst(t *testing.T) {
	mk := func(state task.TaskState) task.Task {
		return task.Task{ID: uuid.NewV7(), StateHistory: []task.StateChange{{State: state}}}
	}
	tasks := []task.Task{
		mk(task.StateCompleted),
		mk(task.StateBlocked),
		mk(task.StateImplement),
		mk(task.StateHumanReview),
		mk(task.StateAbandoned),
	}

	sections := groupTasks(tasks)
	got := make([]string, 0, len(sections))
	for _, s := range sections {
		got = append(got, s.Label)
	}
	want := []string{"Needs attention", "Waiting on review", "In progress", "Done"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("groupTasks() sections = %v, want %v", got, want)
	}
}

func TestSortedLabelsAreStable(t *testing.T) {
	got := sortedLabels(map[string]string{"z": "1", "a": "2", "m": "3"})
	want := []label{{Key: "a", Value: "2"}, {Key: "m", Value: "3"}, {Key: "z", Value: "1"}}
	if len(got) != len(want) {
		t.Fatalf("sortedLabels() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sortedLabels()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestStatusLabels(t *testing.T) {
	if got := statusLabel(task.StateFixVerificationFailure); got != "Fix Verify" {
		t.Errorf("statusLabel() = %q, want %q", got, "Fix Verify")
	}
	if got := statusClass(task.StateBlocked); got != "interrupted" {
		t.Errorf("statusClass(blocked) = %q, want %q", got, "interrupted")
	}
	if got := statusClass(task.StateCompleted); got != "completed" {
		t.Errorf("statusClass(completed) = %q, want %q", got, "completed")
	}
}
