package pipeline

import (
	"strings"
	"testing"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/mhmdkzr/taskman/internal/task"
)

func TestConfigNormalizeDefaultsMaxFixupRounds(t *testing.T) {
	cfg := Config{}.normalize()
	if cfg.MaxFixupRounds != 3 {
		t.Errorf("MaxFixupRounds = %d, want 3", cfg.MaxFixupRounds)
	}

	cfg = Config{MaxFixupRounds: 7}.normalize()
	if cfg.MaxFixupRounds != 7 {
		t.Errorf("MaxFixupRounds = %d, want 7 (explicit value preserved)", cfg.MaxFixupRounds)
	}
}

func testTask() task.Task {
	return task.Task{
		ID:            task.NewTaskID(),
		Title:         "Fix nil pointer",
		TaskType:      "bug",
		Urgency:       "low",
		Importance:    "medium",
		Risk:          "low",
		What:          "Guard against a nil client before calling Send.",
		Why:           "Send panics when the client is nil.",
		How:           "Add a nil check at the top of Send.",
		Where:         []string{"internal/foo/send.go:12"},
		Invariants:    []string{"Send must remain safe to call before Init"},
		CompletedWhen: []string{"tests pass", "vet is clean"},
	}
}

func TestTaskPromptIncludesSpecFields(t *testing.T) {
	prompt := taskPrompt(testTask())
	for _, want := range []string{
		"Fix nil pointer",
		"bug", "low", "medium",
		"Guard against a nil client",
		"Send panics when the client is nil",
		"Add a nil check",
		"internal/foo/send.go:12",
		"Send must remain safe to call before Init",
		"tests pass", "vet is clean",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("taskPrompt missing %q, got:\n%s", want, prompt)
		}
	}
}

func TestTaskPromptOmitsEmptyOptionalSections(t *testing.T) {
	tk := testTask()
	tk.Where = nil
	tk.Invariants = nil
	prompt := taskPrompt(tk)
	if strings.Contains(prompt, "## Where") {
		t.Error("taskPrompt should omit the Where section when empty")
	}
	if strings.Contains(prompt, "## Invariants") {
		t.Error("taskPrompt should omit the Invariants section when empty")
	}
}

func TestFormatDiffsEmpty(t *testing.T) {
	if got := formatDiffs(nil); got != "(no changes)" {
		t.Errorf("formatDiffs(nil) = %q", got)
	}
}

func TestFormatDiffsIncludesEachFile(t *testing.T) {
	diffs := []codebase.Diff{
		{Name: "a.go", ChangeType: codebase.ChangeTypeModified, Additions: 3, Deletions: 1, Patch: "@@ -1 +1,3 @@"},
		{Name: "b.go", ChangeType: codebase.ChangeTypeAdded, Additions: 5, Patch: "@@ -0,0 +1,5 @@"},
	}
	got := formatDiffs(diffs)
	if !strings.Contains(got, "a.go") || !strings.Contains(got, "b.go") {
		t.Errorf("formatDiffs missing a file name, got:\n%s", got)
	}
	if !strings.Contains(got, "+3/-1") || !strings.Contains(got, "+5/-0") {
		t.Errorf("formatDiffs missing addition/deletion counts, got:\n%s", got)
	}
}

func TestFormatDiffsTruncatesLargeOutput(t *testing.T) {
	huge := strings.Repeat("x", maxDiffChars*2)
	diffs := []codebase.Diff{{Name: "big.go", ChangeType: codebase.ChangeTypeModified, Patch: huge}}

	got := formatDiffs(diffs)
	if len(got) > maxDiffChars+200 {
		t.Errorf("formatDiffs length = %d, want capped near %d", len(got), maxDiffChars)
	}
	if !strings.Contains(got, "truncated") {
		t.Error("formatDiffs should note truncation for a diff this large")
	}
}

func TestReviewInputPromptIncludesTaskDiffAndLint(t *testing.T) {
	tk := testTask()
	diffs := []codebase.Diff{{Name: "a.go", ChangeType: codebase.ChangeTypeModified, Additions: 1, Patch: "@@ ... @@"}}
	lint := codebase.LintReport{}

	prompt := reviewInputPrompt(tk, diffs, lint)
	if !strings.Contains(prompt, tk.Title) {
		t.Error("reviewInputPrompt missing task title")
	}
	if !strings.Contains(prompt, "a.go") {
		t.Error("reviewInputPrompt missing diff content")
	}
	if !strings.Contains(prompt, "lint passed") {
		t.Error("reviewInputPrompt missing lint report")
	}
}

func TestStoreToolsConvertsEachField(t *testing.T) {
	tools := []goai.Tool{
		{Name: "read", Description: "reads a file", InputSchema: []byte(`{"type":"object"}`)},
	}
	out := storeTools(tools)
	if len(out) != 1 {
		t.Fatalf("storeTools = %d tools, want 1", len(out))
	}
	if out[0].Name != "read" || out[0].Description != "reads a file" || out[0].InputSchema != `{"type":"object"}` {
		t.Errorf("storeTools[0] = %+v", out[0])
	}
}

// TestRunTaskRequiresLiveModel documents the current testing gap: RunTask
// exercises three real agent turns (execution, review, commit) through
// agent.Options.model, which is only settable via the unexported
// SetModelForTesting hook inside the agent package itself. A fake-model test
// double would need to be exported from internal/agent (or RunTask
// restructured to accept an injectable session factory) before RunTask can
// be covered by a fast unit test; until then this package's coverage is
// limited to its pure helpers (prompt building, diff formatting, tool
// conversion) above.
func TestRunTaskRequiresLiveModel(t *testing.T) {
	t.Skip("RunTask needs a live model or an exported agent test double; see comment above")
}
