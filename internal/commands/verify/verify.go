// Package verify owns the "verify" command: its domain logic and CLI wiring.
package verify

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is verify's input - one reported build-check attempt. Shared
// verbatim by the CLI (cmd.go builds it from flags) and MCP (mcp.go uses it
// as the tool's input type directly) frontends; the json/jsonschema tags
// describe it to MCP clients.
type Request struct {
	ID     string                      `json:"id"               jsonschema:"the task id being verified"`
	Checks map[string]task.CheckResult `json:"checks"           jsonschema:"one result per build check actually run, keyed by check name, each value 'ok' or 'error'"`
	Output string                      `json:"output,omitempty" jsonschema:"combined output from the checks, for a human to read"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	if len(r.Checks) == 0 {
		return fmt.Errorf("at least one check is required")
	}
	return nil
}

// Verify appends one build-check attempt to a task's verifications log. It
// never sets verification.state itself; that only happens via task review
// record's approval.
func Verify(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("verify task: %w", err)
	}
	event := task.VerificationReported{Verification: task.Verification{
		Checks:    req.Checks,
		Output:    req.Output,
		CreatedAt: time.Now().UTC(),
	}}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, event)
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("verify task: %w", err)
	}
	return t, nil
}
