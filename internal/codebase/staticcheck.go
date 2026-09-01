package codebase

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// StaticcheckLocation is a file:line:column location in staticcheck's JSON
// output — a plain object, unlike go vet's "file:line:col" encoded string.
type StaticcheckLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// StaticcheckFinding is one line of `staticcheck -f json` output.
type StaticcheckFinding struct {
	Code     string               `json:"code"`
	Severity string               `json:"severity"`
	Location StaticcheckLocation  `json:"location"`
	End      StaticcheckLocation  `json:"end"`
	Message  string               `json:"message"`
	Related  []StaticcheckRelated `json:"related,omitempty"`
}

// StaticcheckRelated is a secondary position and message attached to a finding.
type StaticcheckRelated struct {
	Location StaticcheckLocation `json:"location"`
	End      StaticcheckLocation `json:"end"`
	Message  string              `json:"message"`
}

// Staticcheck runs staticcheck across the repository and returns its
// findings. A non-empty result is not itself a Go error — staticcheck exits
// non-zero whenever it finds anything, so that exit status alone isn't
// distinguished from a real invocation failure; only unparseable output is.
func (r Repository) Staticcheck() ([]StaticcheckFinding, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	var stderr bytes.Buffer
	cmd := exec.Command("staticcheck", "-f", "json", "./...")
	cmd.Dir = wt.Filesystem.Root()
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("staticcheck: %w", err)
	}

	var findings []StaticcheckFinding
	sc := bufio.NewScanner(stdout)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var f StaticcheckFinding
		if err := json.Unmarshal(line, &f); err != nil {
			return nil, fmt.Errorf("decode staticcheck output: %w", err)
		}
		findings = append(findings, f)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read staticcheck output: %w", err)
	}

	// staticcheck exits non-zero when it reports findings, so Wait's error is
	// only meaningful when we failed to parse anything from stdout at all.
	if err := cmd.Wait(); err != nil && findings == nil {
		return nil, fmt.Errorf("staticcheck: %w: %s", err, stderr.String())
	}
	return findings, nil
}
