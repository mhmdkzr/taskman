// Package create owns the "create" command: its domain logic, CLI wiring,
// and CLI summary output.
package create

import (
	"context"
	"fmt"
	"os"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is create's input - design.md §6. Shared verbatim by the CLI
// (cmd.go builds it from flags) and MCP (mcp.go uses it as the tool's input
// type directly) frontends; the jsonschema tags describe it to MCP clients.
type Request struct {
	Definition    string            `json:"definition"              jsonschema:"what the task should accomplish"`
	ID            string            `json:"id,omitempty"            jsonschema:"use this id instead of generating one"`
	Title         string            `json:"title,omitempty"         jsonschema:"short human-readable title"`
	Labels        map[string]string `json:"labels,omitempty"        jsonschema:"labels as key/value pairs"`
	References    []string          `json:"references,omitempty"    jsonschema:"files or locations relevant to this task"`
	Specification string            `json:"specification,omitempty" jsonschema:"skip straight to implementation by providing the specification up front (requires done_when)"`
	DoneWhen      string            `json:"done_when,omitempty"     jsonschema:"acceptance criteria - required together with specification"`
	Trunk         bool              `json:"trunk,omitempty"         jsonschema:"work this task on the current branch instead of creating a worktree and branch"`
	AutoApprove   bool              `json:"auto_approve,omitempty"  jsonschema:"skip the human review gate - review completes on its own once the commit is made"`
}

func (r Request) validate() error {
	if r.Definition == "" {
		return fmt.Errorf("definition is required")
	}
	if err := task.ValidateLabels(r.Labels); err != nil {
		return fmt.Errorf("validate labels: %w", err)
	}
	return nil
}

// Create makes a new task file, and - the one exception to "taskman
// executes nothing itself" - the task's worktree and branch, guarded by a
// clean-working-tree precondition. With req.Trunk, it skips worktree/branch
// creation and records the repo root and current branch instead, so the
// task is worked in place. See design.md §5.
func Create(ctx context.Context, tasksDir, worktreesDir string, git *task.GitClient, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("create task: %w", err)
	}

	clean, err := git.IsClean(ctx)
	if err != nil {
		return task.Task{}, fmt.Errorf("check working tree: %w", err)
	}
	if !clean {
		return task.Task{}, task.ErrWorkingTreeDirty
	}

	id := req.ID
	if id == "" {
		id = task.GenerateID(req.Title)
	}

	var worktree, branch string
	if req.Trunk {
		worktree, branch, err = git.UseTrunk(ctx)
		if err != nil {
			return task.Task{}, fmt.Errorf("use trunk: %w", err)
		}
	} else {
		worktree, branch, err = git.CreateWorktree(ctx, worktreesDir, id)
		if err != nil {
			return task.Task{}, fmt.Errorf("create worktree: %w", err)
		}
	}

	t := task.Task{
		ID:          id,
		State:       task.StateCreated,
		Title:       req.Title,
		Labels:      req.Labels,
		Definition:  req.Definition,
		References:  req.References,
		AutoApprove: req.AutoApprove,
		Git:         task.Git{Worktree: worktree, Branch: branch, Trunk: req.Trunk},
		Status: task.Status{
			Definition:     task.StageStatus{State: task.StageDone, CompletedAt: new(task.Now())},
			Specification:  task.StageStatus{State: task.StagePending},
			Implementation: task.StageStatus{State: task.StagePending},
			Verification:   task.StageStatus{State: task.StagePending},
			Review:         task.StageStatus{State: task.StagePending},
			Merge:          task.StageStatus{State: task.StagePending},
		},
	}

	if req.Specification != "" && req.DoneWhen != "" {
		t.Specification = req.Specification
		t.DoneWhen = req.DoneWhen
		t.Status.Specification = task.StageStatus{State: task.StageDone, CompletedAt: new(task.Now())}
		t.State = task.StateStarted
	}

	if t.ID == "" {
		return task.Task{}, fmt.Errorf("create task: id is required")
	}
	if err := os.MkdirAll(tasksDir, 0o700); err != nil {
		return task.Task{}, fmt.Errorf("create tasks dir: %w", err)
	}
	if err := task.WriteTaskFile(tasksDir, t); err != nil {
		return task.Task{}, fmt.Errorf("write task file: %w", err)
	}
	return t, nil
}
