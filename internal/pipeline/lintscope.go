package pipeline

import (
	"fmt"
	"path/filepath"

	"github.com/mhmdkzr/taskman/internal/codebase"
)

// scopedLint runs the repository's full lint pass and filters it down to
// findings in files the task has actually changed (see
// scopeLintToChangedFiles) — the shape every pipeline call site wants,
// rather than the raw whole-repository report.
func scopedLint(repo codebase.Repository) (codebase.LintReport, error) {
	root, err := repo.Root()
	if err != nil {
		return codebase.LintReport{}, fmt.Errorf("root: %w", err)
	}
	diffs, err := repo.Diff()
	if err != nil {
		return codebase.LintReport{}, fmt.Errorf("diff: %w", err)
	}
	report, err := repo.Lint()
	if err != nil {
		return codebase.LintReport{}, fmt.Errorf("lint: %w", err)
	}
	return scopeLintToChangedFiles(report, root, diffs), nil
}

// scopeLintToChangedFiles filters a LintReport down to findings in files the
// task actually changed. Lint runs `go vet`/staticcheck/golangci-lint across
// the whole repository — there is no cheap way to ask any of them to check
// only a diff — so a repo with any pre-existing lint debt would otherwise
// hand the executor hundreds of findings unrelated to its task on every
// single run, which it can neither fix in a bounded number of rounds nor
// safely ignore (nothing distinguishes "pre-existing" from "caused by your
// change" in the raw report). Scoping to the diff is what makes lint
// findings an actionable, convergent signal for the specific change being
// made instead of a standing tax on every task run against an imperfect
// codebase.
//
// A finding in a file the task didn't touch is dropped even if it's a
// regression the change indirectly caused elsewhere (rare for isolated
// tasks, and out of scope for this pass) — see the "adjustments" note this
// case prompted for the tradeoff.
func scopeLintToChangedFiles(report codebase.LintReport, root string, diffs []codebase.Diff) codebase.LintReport {
	changed := make(map[string]bool, len(diffs))
	for _, d := range diffs {
		changed[d.Name] = true
	}

	var scoped codebase.LintReport
	for _, d := range report.VetIssues {
		if changed[relLintPath(root, d.Posn.File)] {
			scoped.VetIssues = append(scoped.VetIssues, d)
		}
	}
	for _, f := range report.StaticcheckIssues {
		if changed[relLintPath(root, f.Location.File)] {
			scoped.StaticcheckIssues = append(scoped.StaticcheckIssues, f)
		}
	}
	for _, i := range report.LintIssues {
		if changed[relLintPath(root, i.Pos.Filename)] {
			scoped.LintIssues = append(scoped.LintIssues, i)
		}
	}
	return scoped
}

// relLintPath normalizes a file path a lint tool reported (go vet's are
// absolute; golangci-lint's are already worktree-relative; staticcheck's
// have been observed as both) to worktree-relative, matching
// codebase.Diff.Name's format, so both can be compared directly.
func relLintPath(root, file string) string {
	if !filepath.IsAbs(file) {
		return file
	}
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return file
	}
	return rel
}
