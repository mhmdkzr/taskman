package codebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// GolangciLintPosition is a file:offset:line:column location in
// golangci-lint's JSON output.
type GolangciLintPosition struct {
	Filename string `json:"Filename"`
	Offset   int    `json:"Offset"`
	Line     int    `json:"Line"`
	Column   int    `json:"Column"`
}

// GolangciLintIssue is one entry in golangci-lint's JSON "Issues" array.
type GolangciLintIssue struct {
	FromLinter  string               `json:"FromLinter"`
	Text        string               `json:"Text"`
	Severity    string               `json:"Severity"`
	SourceLines []string             `json:"SourceLines,omitempty"`
	Pos         GolangciLintPosition `json:"Pos"`
}

// GolangciLint runs golangci-lint across the repository using its own
// discovered config (.golangci.yaml, etc.) and returns its issues. A
// non-empty result is not itself a Go error — golangci-lint exits non-zero
// whenever it finds anything, so that exit status alone isn't distinguished
// from a real invocation failure; only unparseable output is.
func (r Repository) GolangciLint() ([]GolangciLintIssue, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("golangci-lint", "run",
		"--output.json.path", "stdout", "--show-stats=false", "./...")
	cmd.Dir = wt.Filesystem.Root()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if stdout.Len() == 0 {
		if runErr != nil {
			return nil, fmt.Errorf("golangci-lint: %w: %s", runErr, stderr.String())
		}
		return nil, nil
	}

	var result struct {
		Issues []GolangciLintIssue `json:"Issues"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("decode golangci-lint output: %w", err)
	}
	return result.Issues, nil
}
