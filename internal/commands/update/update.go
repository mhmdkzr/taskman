// Package update owns the "update" command: its domain logic and CLI wiring.
package update

import (
	"fmt"
	"maps"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is update's input - patch semantics, only non-nil/given fields
// are applied. Shared verbatim by the CLI (cmd.go builds it from flags) and
// MCP (mcp.go uses it as the tool's input type directly) frontends; the
// json/jsonschema tags describe it to MCP clients.
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
// specification) or the rest of Git/workflow state (taskman-managed). Most
// metadata has no workflow-state precondition; auto-approve is the exception,
// since State.AutoApproveMoot states can no longer act on it.
func Update(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("update task: %w", err)
	}
	merged := make(map[string]string)
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		t := current.Clone()
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
				return task.Task{}, fmt.Errorf("validate labels: %w", err)
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
			if current.State.AutoApproveMoot() {
				return task.Task{}, fmt.Errorf(
					"update task: %w: state is %q", task.ErrAutoApproveTooLate, current.State,
				)
			}
			t.AutoApprove = *req.AutoApprove
		}
		return t, nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("update task: %w", err)
	}
	return t, nil
}
