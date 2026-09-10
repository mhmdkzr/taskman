// Package record owns the "reviewed" command: its domain logic and
// CLI wiring.
package record

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is reviewed's input - one automated review round's verdict.
// Shared verbatim by the CLI (cmd.go builds it from flags) and MCP (mcp.go
// uses it as the tool's input type directly) frontends; the json/jsonschema
// tags describe it to MCP clients.
type Request struct {
	ID       string         `json:"id"                 jsonschema:"the task id being reviewed"`
	Approved bool           `json:"approved"           jsonschema:"whether the automated review approved this attempt"`
	Findings []task.Finding `json:"findings,omitempty" jsonschema:"findings against the review, each with a file and its full detail text"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// RecordReview reports the automated review round's verdict inside
// verification. Approval advances the task to the review stage; rejection
// either loops back for another attempt (attempt 1) or blocks the task
// (attempt 2, the fixed two-round cap).
func RecordReview(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("record review: %w", err)
	}
	event := task.AutomatedReviewRecorded{
		Approved: req.Approved, Findings: req.Findings, At: time.Now().UTC(),
	}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, event)
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("record review: %w", err)
	}
	return t, nil
}
