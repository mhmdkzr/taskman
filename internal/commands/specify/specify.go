// Package specify owns the "specify" command: its domain logic, CLI
// wiring, and dispatch prompt.
package specify

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is specify's input. Shared verbatim by the CLI (cmd.go builds it
// from flags) and MCP (mcp.go uses it as the tool's input type directly)
// frontends; the json/jsonschema tags describe it to MCP clients.
type Request struct {
	ID       string `json:"id"        jsonschema:"the task id to specify"`
	Result   string `json:"result"    jsonschema:"the drafted specification"`
	DoneWhen string `json:"done_when" jsonschema:"the drafted acceptance criteria"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	if r.Result == "" {
		return fmt.Errorf("result is required")
	}
	if r.DoneWhen == "" {
		return fmt.Errorf("done_when is required")
	}
	return nil
}

// Specify writes a task's specification and done_when, drafted from its
// definition - design.md §6.
func Specify(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("specify task: %w", err)
	}
	t, err := task.MutateTask(tasksDir, req.ID, func(t *task.Task) error {
		if t.Status.Specification.State == task.StageDone {
			return task.NotInState("specification", string(t.Status.Specification.State), "!= done")
		}
		t.Specification = req.Result
		t.DoneWhen = req.DoneWhen
		t.Status.Specification = task.StageStatus{State: task.StageDone, CompletedAt: new(task.Now())}
		t.State = task.StateStarted
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("specify task: %w", err)
	}
	return t, nil
}
