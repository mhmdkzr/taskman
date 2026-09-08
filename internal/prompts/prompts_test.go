package prompts

import (
	"strings"
	"testing"
)

func TestSpecify(t *testing.T) {
	got := Specify{Definition: "Fix the bug", References: []string{"a.go:1", "b.go:2"}}.Render()
	for _, want := range []string{"Fix the bug", "a.go:1", "b.go:2"} {
		if !strings.Contains(got, want) {
			t.Errorf("Specify.Render() missing %q, got:\n%s", want, got)
		}
	}
}

func TestImplement(t *testing.T) {
	got := Implement{Definition: "def", Specification: "spec", DoneWhen: "criteria"}.Render()
	for _, want := range []string{"def", "spec", "criteria"} {
		if !strings.Contains(got, want) {
			t.Errorf("Implement.Render() missing %q, got:\n%s", want, got)
		}
	}
}

func TestFix(t *testing.T) {
	got := Fix{Specification: "spec", DoneWhen: "criteria", Reason: "build check failed: boom"}.Render()
	if !strings.Contains(got, "boom") {
		t.Errorf("Fix.Render() missing reason, got:\n%s", got)
	}
}

func TestReview(t *testing.T) {
	got := Review{Specification: "spec", DoneWhen: "criteria"}.Render()
	if !strings.Contains(got, "spec") || !strings.Contains(got, "criteria") {
		t.Errorf("Review.Render() missing content, got:\n%s", got)
	}
}

func TestCommit(t *testing.T) {
	got := Commit{Title: "Fix Doc Drift", Specification: "spec"}.Render()
	if !strings.Contains(got, "Fix Doc Drift") {
		t.Errorf("Commit.Render() missing title, got:\n%s", got)
	}
}

func TestDispatchWrapper(t *testing.T) {
	got := DispatchWrapper{
		Body:       "BODY TEXT",
		Worktree:   ".worktrees/abc",
		Branch:     "task/abc",
		ReportWith: "task verify abc",
	}.Render()
	for _, want := range []string{"BODY TEXT", ".worktrees/abc", "task/abc", "task verify abc"} {
		if !strings.Contains(got, want) {
			t.Errorf("DispatchWrapper.Render() missing %q, got:\n%s", want, got)
		}
	}
}

func TestRunVerifyAndRunMerge(t *testing.T) {
	v := RunVerify{TaskID: "abc", Worktree: ".worktrees/abc", Branch: "task/abc"}.Render()
	if !strings.Contains(v, "abc") || !strings.Contains(v, "task verify abc") {
		t.Errorf("RunVerify.Render() = %q", v)
	}
	m := RunMerge{TaskID: "abc", Worktree: ".worktrees/abc", Branch: "task/abc"}.Render()
	if !strings.Contains(m, "task merge abc") {
		t.Errorf("RunMerge.Render() = %q", m)
	}
}

func TestWaitHumanReviewAndWaitBlocked(t *testing.T) {
	w := WaitHumanReview{
		TaskID:       "abc",
		Title:        "T",
		Attempt:      2,
		CommitHash:   "deadbeef",
		Branch:       "task/abc",
		Worktree:     ".worktrees/abc",
		WaitingSince: "2026-01-01 00:00 UTC",
	}.Render()
	for _, want := range []string{"abc", "T", "2", "deadbeef", "task/abc", ".worktrees/abc", "2026-01-01"} {
		if !strings.Contains(w, want) {
			t.Errorf("WaitHumanReview.Render() missing %q, got:\n%s", want, w)
		}
	}

	b := WaitBlocked{
		TaskID:       "abc",
		Title:        "T",
		Stage:        "verification",
		WaitingSince: "now",
		Reason:       "rejected twice",
		Worktree:     ".worktrees/abc",
		Branch:       "task/abc",
	}.Render()
	for _, want := range []string{"verification", "rejected twice"} {
		if !strings.Contains(b, want) {
			t.Errorf("WaitBlocked.Render() missing %q, got:\n%s", want, b)
		}
	}
}

func TestDoneMergedAndDoneAbandoned(t *testing.T) {
	m := DoneMerged{TaskID: "abc", Title: "T", CommitHash: "deadbeef"}.Render()
	if !strings.Contains(m, "deadbeef") {
		t.Errorf("DoneMerged.Render() missing hash, got:\n%s", m)
	}
	a := DoneAbandoned{TaskID: "abc", Title: "T", Reason: "superseded"}.Render()
	if !strings.Contains(a, "superseded") {
		t.Errorf("DoneAbandoned.Render() missing reason, got:\n%s", a)
	}
}

func TestCreateSummaryAndTaskSummary(t *testing.T) {
	c := CreateSummary{TaskID: "abc", Title: "T", Worktree: ".worktrees/abc", Branch: "task/abc"}.Render()
	for _, want := range []string{"abc", "T", ".worktrees/abc", "task/abc"} {
		if !strings.Contains(c, want) {
			t.Errorf("CreateSummary.Render() missing %q, got:\n%s", want, c)
		}
	}

	s := TaskSummary{TaskID: "abc", Title: "T", State: "started", Stage: "awaiting merge"}.Render()
	for _, want := range []string{"abc", "T", "started", "awaiting merge"} {
		if !strings.Contains(s, want) {
			t.Errorf("TaskSummary.Render() missing %q, got:\n%s", want, s)
		}
	}
}

func TestRenderTrimsTrailingNewline(t *testing.T) {
	got := DoneAbandoned{TaskID: "abc", Title: "T", Reason: "x"}.Render()
	if len(got) > 0 && got[len(got)-1] == '\n' {
		t.Errorf("Render() left a trailing newline: %q", got)
	}
}
