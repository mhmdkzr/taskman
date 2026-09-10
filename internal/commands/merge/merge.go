// Package merge owns the "merged" command: its domain logic and CLI wiring.
package merge

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is merge's input. Shared verbatim by the CLI (cmd.go builds it
// from flags) and MCP (mcp.go uses it as the tool's input type directly)
// frontends; the json/jsonschema tags describe it to MCP clients.
type Request struct {
	ID     string `json:"id"               jsonschema:"the task id whose branch was merged"`
	Commit string `json:"commit,omitempty" jsonschema:"override the recorded commit hash, e.g. after a non-fast-forward merge"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Merge records that the caller already merged the task's branch, then
// commits the task's own now-terminal file itself - see
// gitclient.RecordBookkeeping. Never reached for a trunk task, which reaches
// completed directly when its review is approved.
func Merge(ctx context.Context, tasksDir string, gitClient *git.Client, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("merge task: %w", err)
	}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, task.MergeCompleted{CommitOverride: req.Commit})
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("merge task: %w", err)
	}
	if err := git.RecordBookkeeping(ctx, gitClient, tasksDir, t, "completion"); err != nil {
		return task.Task{}, fmt.Errorf("merge task: %w", err)
	}
	return t, nil
}
