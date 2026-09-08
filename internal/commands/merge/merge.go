// Package merge owns the "merge" command: its domain logic and CLI wiring.
package merge

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
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

// Merge records that the caller already merged the task's branch.
func Merge(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("merge task: %w", err)
	}
	t, err := task.MutateTask(tasksDir, req.ID, func(t *task.Task) error {
		if t.Status.Review.State != task.StageDone {
			return task.NotInState("review", string(t.Status.Review.State), "done")
		}
		t.Status.Merge = task.StageStatus{State: task.StageDone, CompletedAt: new(task.Now())}
		t.State = task.StateCompleted
		if req.Commit != "" && t.Git.Commit != nil {
			t.Git.Commit.Hash = req.Commit
		}
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("merge task: %w", err)
	}
	return t, nil
}
