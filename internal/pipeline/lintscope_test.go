package pipeline

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/codebase"
)

func TestScopeLintToChangedFilesDropsUnrelatedFindings(t *testing.T) {
	root := "/repo"
	report := codebase.LintReport{
		VetIssues: []codebase.VetDiagnostic{
			{
				Posn:    codebase.Position{File: "/repo/internal/foo/foo.go", Line: 1, Column: 1},
				Message: "changed file, absolute path",
			},
			{
				Posn:    codebase.Position{File: "/repo/internal/bar/bar.go", Line: 2, Column: 1},
				Message: "unrelated file",
			},
		},
		StaticcheckIssues: []codebase.StaticcheckFinding{
			{
				Location: codebase.StaticcheckLocation{File: "/repo/internal/foo/foo.go", Line: 3},
				Message:  "changed file",
			},
			{
				Location: codebase.StaticcheckLocation{File: "/repo/internal/baz/baz.go", Line: 4},
				Message:  "unrelated file",
			},
		},
		LintIssues: []codebase.GolangciLintIssue{
			{
				Pos:  codebase.GolangciLintPosition{Filename: "internal/foo/foo.go", Line: 5},
				Text: "changed file, relative path",
			},
			{
				Pos:  codebase.GolangciLintPosition{Filename: "internal/qux/qux.go", Line: 6},
				Text: "unrelated file",
			},
		},
	}
	diffs := []codebase.Diff{
		{Name: "internal/foo/foo.go", ChangeType: codebase.ChangeTypeModified},
	}

	scoped := scopeLintToChangedFiles(report, root, diffs)

	if len(scoped.VetIssues) != 1 || scoped.VetIssues[0].Message != "changed file, absolute path" {
		t.Errorf("VetIssues = %+v, want only the foo.go finding", scoped.VetIssues)
	}
	if len(scoped.StaticcheckIssues) != 1 || scoped.StaticcheckIssues[0].Message != "changed file" {
		t.Errorf("StaticcheckIssues = %+v, want only the foo.go finding", scoped.StaticcheckIssues)
	}
	if len(scoped.LintIssues) != 1 || scoped.LintIssues[0].Text != "changed file, relative path" {
		t.Errorf("LintIssues = %+v, want only the foo.go finding", scoped.LintIssues)
	}
}

func TestScopeLintToChangedFilesEmptyDiffDropsEverything(t *testing.T) {
	report := codebase.LintReport{
		VetIssues: []codebase.VetDiagnostic{
			{Posn: codebase.Position{File: "/repo/internal/foo/foo.go", Line: 1}},
		},
	}
	scoped := scopeLintToChangedFiles(report, "/repo", nil)
	if !scoped.Clean() {
		t.Errorf("scoped = %+v, want clean when nothing changed", scoped)
	}
}

func TestRelLintPath(t *testing.T) {
	cases := []struct {
		root, file, want string
	}{
		{"/repo", "/repo/internal/foo/foo.go", "internal/foo/foo.go"},
		{"/repo", "internal/foo/foo.go", "internal/foo/foo.go"},
		{"/repo", "/other/internal/foo/foo.go", "../other/internal/foo/foo.go"},
	}
	for _, c := range cases {
		if got := relLintPath(c.root, c.file); got != c.want {
			t.Errorf("relLintPath(%q, %q) = %q, want %q", c.root, c.file, got, c.want)
		}
	}
}
