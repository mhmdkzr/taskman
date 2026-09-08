// Package create owns the "create" command: its domain logic, CLI wiring,
// and CLI summary output.
package create

import (
	"context"
	"fmt"
	"os"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is task create's input - design.md §6.
type Request struct {
	Definition    string
	ID            string
	Title         string
	Labels        map[string]string
	References    []string
	Specification string
	DoneWhen      string
}

// Create makes a new task file, and - the one exception to "taskman
// executes nothing itself" - the task's worktree and branch, guarded by a
// clean-working-tree precondition. See design.md §5.
func Create(ctx context.Context, tasksDir, worktreesDir string, git *task.GitClient, req Request) (task.Task, error) {
	if req.Definition == "" {
		return task.Task{}, fmt.Errorf("create task: definition is required")
	}
	if err := task.ValidateLabels(req.Labels); err != nil {
		return task.Task{}, fmt.Errorf("validate labels: %w", err)
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

	worktree, branch, err := git.CreateWorktree(ctx, worktreesDir, id)
	if err != nil {
		return task.Task{}, fmt.Errorf("create worktree: %w", err)
	}

	t := task.Task{
		ID:         id,
		State:      task.StateCreated,
		Title:      req.Title,
		Labels:     req.Labels,
		Definition: req.Definition,
		References: req.References,
		Git:        task.Git{Worktree: worktree, Branch: branch},
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
