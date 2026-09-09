// Package update owns the "update" command: its domain logic and CLI wiring.
package update

import (
	"fmt"
	"maps"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is update's input - patch semantics, only non-nil/given fields
// are applied. Shared verbatim by the CLI (cmd.go builds it from flags) and
// MCP (mcp.go uses it as the tool's input type directly) frontends; the
// json/jsonschema tags describe it to MCP clients. design.md §6.
type Request struct {
	ID              string            `json:"id"                         jsonschema:"the task id to patch"`
	Title           *string           `json:"title,omitempty"            jsonschema:"new title"`
	SetLabels       map[string]string `json:"set_labels,omitempty"       jsonschema:"labels to set as key/value pairs"`
	UnsetLabels     []string          `json:"unset_labels,omitempty"     jsonschema:"label keys to remove"`
	References      []string          `json:"references,omitempty"       jsonschema:"replace the reference list"`
	ClearReferences bool              `json:"clear_references,omitempty" jsonschema:"remove every reference"`
	Trunk           *bool             `json:"trunk,omitempty"            jsonschema:"set or unset whether the task works on the current branch instead of an isolated worktree/branch"`
	AutoApprove     *bool             `json:"auto_approve,omitempty"     jsonschema:"set or unset whether the task's review stage completes on its own without a human gate"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Update patches a task's metadata (title, labels, references, trunk,
// auto-approve). It never touches specification/done_when (own command:
// specify) or the rest of git/status (taskman-managed). No precondition on
// State/Status - metadata isn't workflow state. design.md §6.
func Update(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("update task: %w", err)
	}
	merged := make(map[string]string)
	t, err := task.MutateTask(tasksDir, req.ID, func(t *task.Task) error {
		if req.Title != nil {
			t.Title = *req.Title
		}
		if len(req.SetLabels) > 0 || len(req.UnsetLabels) > 0 {
			maps.Copy(merged, t.Labels)
			maps.Copy(merged, req.SetLabels)
			for _, k := range req.UnsetLabels {
				delete(merged, k)
			}
			if err := task.ValidateLabels(merged); err != nil {
				return fmt.Errorf("validate labels: %w", err)
			}
			t.Labels = merged
		}
		if req.ClearReferences {
			t.References = nil
		} else if len(req.References) > 0 {
			t.References = req.References
		}
		if req.Trunk != nil {
			t.Git.Trunk = *req.Trunk
		}
		if req.AutoApprove != nil {
			t.AutoApprove = *req.AutoApprove
		}
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("update task: %w", err)
	}
	return t, nil
}
