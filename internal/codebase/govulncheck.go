package codebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// GovulncheckMessage is one entry in govulncheck's JSON stream. Exactly one
// field is set per message; OSV is left as raw JSON since its schema (an
// OSV.dev vulnerability report) lives in an internal x/vuln package this
// repository can't import.
type GovulncheckMessage struct {
	Config   *GovulncheckConfig   `json:"config,omitempty"`
	Progress *GovulncheckProgress `json:"progress,omitempty"`
	OSV      json.RawMessage      `json:"osv,omitempty"`
	Finding  *GovulncheckFinding  `json:"finding,omitempty"`
}

// GovulncheckConfig is the stream's first message, describing the scan itself.
type GovulncheckConfig struct {
	ProtocolVersion string `json:"protocol_version"`
	ScannerName     string `json:"scanner_name,omitempty"`
	ScannerVersion  string `json:"scanner_version,omitempty"`
	DB              string `json:"db,omitempty"`
	GoVersion       string `json:"go_version,omitempty"`
	ScanLevel       string `json:"scan_level,omitempty"`
	ScanMode        string `json:"scan_mode,omitempty"`
}

// GovulncheckProgress is an informational status update; safe to ignore.
type GovulncheckProgress struct {
	Message string `json:"message,omitempty"`
}

// GovulncheckFinding reports a vulnerability actually reachable from this
// module, as opposed to one merely present in the build graph.
type GovulncheckFinding struct {
	OSV          string             `json:"osv,omitempty"`
	FixedVersion string             `json:"fixed_version,omitempty"`
	Trace        []GovulncheckFrame `json:"trace,omitempty"`
}

// GovulncheckFrame is one call-stack entry in a finding's trace, ordered from
// the vulnerable symbol to the entry point.
type GovulncheckFrame struct {
	Module   string `json:"module"`
	Version  string `json:"version,omitempty"`
	Package  string `json:"package,omitempty"`
	Function string `json:"function,omitempty"`
	Receiver string `json:"receiver,omitempty"`
}

// Govulncheck runs govulncheck across the repository and returns every
// finding — a reachable vulnerability, not merely a vulnerable dependency in
// the build graph. Unlike Staticcheck/GolangciLint, a non-zero exit here is
// always a real invocation error (network/build failure, etc.): with -json
// output govulncheck exits 0 regardless of how many vulnerabilities it
// finds, so the findings themselves are never reflected in the exit code.
func (r Repository) Govulncheck() ([]GovulncheckFinding, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %w", err)
	}

	var stderr bytes.Buffer
	cmd := exec.Command("govulncheck", "-json", "./...")
	cmd.Dir = wt.Filesystem.Root()
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("govulncheck: %w", err)
	}

	var findings []GovulncheckFinding
	dec := json.NewDecoder(stdout)
	for dec.More() {
		var msg GovulncheckMessage
		if err := dec.Decode(&msg); err != nil {
			// Reap the process so it doesn't leak; its own error (if any) is
			// secondary to the decode failure we're already returning.
			if waitErr := cmd.Wait(); waitErr != nil {
				return nil, fmt.Errorf("decode govulncheck output: %w (process also failed: %w)", err, waitErr)
			}
			return nil, fmt.Errorf("decode govulncheck output: %w", err)
		}
		if msg.Finding != nil {
			findings = append(findings, *msg.Finding)
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("govulncheck: %w: %s", err, stderr.String())
	}
	return findings, nil
}
