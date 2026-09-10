// Package reject owns the "review reject" command: its domain logic and
// CLI wiring.
package reject

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is review reject's input. Shared verbatim by the CLI (cmd.go
// builds it from flags) and MCP (mcp.go uses it as the tool's input type
// directly) frontends; the json/jsonschema tags describe it to MCP clients.
type Request struct {
	ID     string `json:"id"     jsonschema:"the task id being rejected"`
	Reason string `json:"reason" jsonschema:"why the review was rejected"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	if r.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

// RejectReview records a human's rejection and starts review-reject
// recovery.
func RejectReview(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("reject review: %w", err)
	}
	event := task.HumanReviewRejected{Reason: req.Reason, At: time.Now().UTC()}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, event)
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("reject review: %w", err)
	}
	return t, nil
}
