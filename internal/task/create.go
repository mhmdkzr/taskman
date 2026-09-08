package task

import (
	"context"
	"fmt"
)

// CreateRequest is task create's input - design.md §6.
type CreateRequest struct {
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
func Create(ctx context.Context, repo *Repo, git *GitClient, worktreesDir string, req CreateRequest) (Task, error) {
	if req.Definition == "" {
		return Task{}, fmt.Errorf("create task: definition is required")
	}
	if err := ValidateLabels(req.Labels); err != nil {
		return Task{}, err
	}

	clean, err := git.IsClean(ctx)
	if err != nil {
		return Task{}, err
	}
	if !clean {
		return Task{}, ErrWorkingTreeDirty
	}

	id := req.ID
	if id == "" {
		id = GenerateID(req.Title)
	}

	worktree, branch, err := git.CreateWorktree(ctx, worktreesDir, id)
	if err != nil {
		return Task{}, err
	}

	t := Task{
		ID:         id,
		State:      StateCreated,
		Title:      req.Title,
		Labels:     req.Labels,
		Definition: req.Definition,
		References: req.References,
		Git:        Git{Worktree: worktree, Branch: branch},
		Status: Status{
			Definition:     StageStatus{State: StageDone, CompletedAt: new(now())},
			Specification:  StageStatus{State: StagePending},
			Implementation: StageStatus{State: StagePending},
			Verification:   StageStatus{State: StagePending},
			Review:         StageStatus{State: StagePending},
			Merge:          StageStatus{State: StagePending},
		},
	}

	if req.Specification != "" && req.DoneWhen != "" {
		t.Specification = req.Specification
		t.DoneWhen = req.DoneWhen
		t.Status.Specification = StageStatus{State: StageDone, CompletedAt: new(now())}
		t.State = StateStarted
	}

	if err := repo.Create(t); err != nil {
		return Task{}, err
	}
	return t, nil
}
