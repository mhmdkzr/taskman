package codebase

import (
	"fmt"
	"strings"
)

// LintReport is the combined result of go vet, staticcheck, and
// golangci-lint — the pipeline runs all three automatically after an agent
// claims a task is done, and feeds the report back to it as the next turn's
// input. govulncheck is deliberately excluded — it's network-dependent and
// slow, a candidate for a separate, less frequent check rather than running
// on every task.
type LintReport struct {
	VetIssues         []VetDiagnostic
	StaticcheckIssues []StaticcheckFinding
	LintIssues        []GolangciLintIssue
}

// Clean reports whether every check found nothing.
func (l LintReport) Clean() bool {
	return len(l.VetIssues) == 0 && len(l.StaticcheckIssues) == 0 && len(l.LintIssues) == 0
}

// maxLintReportChars bounds String's output so a file with an unusually
// large number of findings (or one very verbose message) cannot flood an
// agent's context window.
const maxLintReportChars = 20000

// String renders the report as plain text suitable for feeding back to an
// agent as its next turn's input.
func (l LintReport) String() string {
	if l.Clean() {
		return "lint passed: no issues from go vet, staticcheck, or golangci-lint"
	}
	var b strings.Builder
	if len(l.VetIssues) > 0 {
		b.WriteString("go vet:\n")
		for _, d := range l.VetIssues {
			fmt.Fprintf(&b, "  %s: %s\n", d.Posn, d.Message)
		}
		b.WriteString("\n")
	}
	if len(l.StaticcheckIssues) > 0 {
		b.WriteString("staticcheck:\n")
		for _, f := range l.StaticcheckIssues {
			fmt.Fprintf(&b, "  %s:%d:%d: %s [%s]\n", f.Location.File, f.Location.Line, f.Location.Column, f.Message, f.Code)
		}
		b.WriteString("\n")
	}
	if len(l.LintIssues) > 0 {
		b.WriteString("golangci-lint:\n")
		for _, i := range l.LintIssues {
			fmt.Fprintf(&b, "  %s:%d:%d: %s [%s]\n", i.Pos.Filename, i.Pos.Line, i.Pos.Column, i.Text, i.FromLinter)
		}
	}
	out := strings.TrimRight(b.String(), "\n")
	if len(out) > maxLintReportChars {
		out = out[:maxLintReportChars] + fmt.Sprintf("\n... (truncated to %d chars)", maxLintReportChars)
	}
	return out
}

// Lint runs go vet, staticcheck, and golangci-lint and collects their
// findings into a LintReport.
//
// Lint only returns an error for something that stops it from running at
// all (a tool missing from PATH, a real invocation failure). Findings from
// any individual check are never an error — they're reported through
// LintReport, so a task with findings still gets a normal report back
// instead of an opaque failure.
func (r Repository) Lint() (LintReport, error) {
	var report LintReport

	vetTree, err := r.GoVet()
	if err != nil {
		return LintReport{}, fmt.Errorf("lint: go vet: %w", err)
	}
	for _, analyzers := range vetTree {
		for _, res := range analyzers {
			report.VetIssues = append(report.VetIssues, res.Diagnostics...)
		}
	}

	staticcheckFindings, err := r.Staticcheck()
	if err != nil {
		return LintReport{}, fmt.Errorf("lint: staticcheck: %w", err)
	}
	report.StaticcheckIssues = staticcheckFindings

	lintIssues, err := r.GolangciLint()
	if err != nil {
		return LintReport{}, fmt.Errorf("lint: golangci-lint: %w", err)
	}
	report.LintIssues = lintIssues

	return report, nil
}
