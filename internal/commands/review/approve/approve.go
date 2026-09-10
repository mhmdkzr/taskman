// Package approve owns the "review approved" command: its domain logic and
// CLI wiring.
package approve

import (
	"context"
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is review approved's input. Shared verbatim by the CLI (cmd.go
// builds it from flags) and MCP (mcp.go uses it as the tool's input type
// directly) frontends; the json/jsonschema tags describe it to MCP clients.
type Request struct {
	ID      string `json:"id"                jsonschema:"the task id being approved"`
	Comment string `json:"comment,omitempty" jsonschema:"an optional note, e.g. LGTM"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// ApproveReview records a human's approval at the review stage. A trunk
// task transitions directly to completed, so this also commits its
// now-terminal task file through the Git shell.
func ApproveReview(ctx context.Context, tasksDir string, gitClient *git.Client, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("approve review: %w", err)
	}
	event := task.HumanReviewApproved{Comment: req.Comment, At: time.Now().UTC()}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, event)
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("approve review: %w", err)
	}
	if t.State == task.StateCompleted {
		if err := git.RecordBookkeeping(ctx, gitClient, tasksDir, t, "completion"); err != nil {
			return task.Task{}, fmt.Errorf("approve review: %w", err)
		}
	}
	return t, nil
}
