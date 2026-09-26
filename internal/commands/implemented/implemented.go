// Package implemented owns the "implemented" command: it reports a task's
// implementation as complete. All requirement/policy for the implementation
// (verification checks, review gates, worktree/branch) is decided earlier, at
// `specified` time - this command only reports the fact that an
// implementation matching that policy now exists.
package implemented

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is implemented's input: just the task's identity. Everything else
// this event needs (verification/review policy, worktree/branch) is read
// back from the task's own Specification.
type Request struct {
	ID uuid.UUID `json:"id" jsonschema:"the task that was implemented"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Implemented reports req's implementation as complete and returns the
// resulting task. The recorded Implementation's Verification and Review
// policy, and its Git worktree/branch, are copied forward from the task's
// Specification (set by `specified`) rather than accepted as input here -
// see Request's doc comment.
func Implemented(ctx context.Context, st *store.Store, gitClient *git.Client, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("record implementation: %w", err)
	}
	current, err := st.Read(ctx, req.ID)
	if err != nil {
		return task.Task{}, fmt.Errorf("record implementation: %w", err)
	}
	if current.Specification == nil {
		return task.Task{}, fmt.Errorf("record implementation: specification not yet reported")
	}
	spec := current.Specification

	if isUnspecifiedPolicy(*spec) {
		return task.Task{}, fmt.Errorf(
			"record implementation: %s was specified before implementation policy moved to `specified` - "+
				"call `specified` again to declare verification, implementation-review, and worktree policy first",
			req.ID,
		)
	}

	gitLocation, err := resolveGit(ctx, gitClient, spec.Worktree)
	if err != nil {
		return task.Task{}, fmt.Errorf("record implementation: %w", err)
	}

	event := task.ImplementationCompleted{
		Implementation: task.Implementation{
			Git:          gitLocation,
			Verification: spec.Verification,
			Review:       spec.ImplementationReview,
		},
		At: time.Now().UTC(),
	}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("record implementation: %w", err)
	}
	return t, nil
}

// isUnspecifiedPolicy reports whether spec declares no implementation
// policy at all - the signature of a task specified before this policy
// moved to `specified`, which implemented refuses to guess at. Verification
// and ReviewConfiguration both hold slice fields (Attempts/Unblocks,
// Results), so they aren't comparable with == - each policy area is checked
// field by field instead, the same way Implementation.validate() already
// has to.
func isUnspecifiedPolicy(spec task.Specification) bool {
	noVerification := !spec.Verification.Tests.Unit &&
		!spec.Verification.Tests.Integration &&
		!spec.Verification.Tests.EndToEnd &&
		!spec.Verification.Linters
	noImplReview := !spec.ImplementationReview.Agent.Required && !spec.ImplementationReview.Human.Required
	return noVerification && noImplReview && !spec.Worktree.UseWorktree
}

// resolveGit returns the Git location to record for this implementation: the
// planned worktree/branch if the specification called for a fresh one, or
// the current repository's own worktree/branch, auto-detected, otherwise.
func resolveGit(ctx context.Context, gitClient *git.Client, policy task.WorktreePolicy) (task.Git, error) {
	if policy.UseWorktree {
		return task.Git{Worktree: policy.Worktree, Branch: policy.Branch}, nil
	}
	worktree, branch, err := gitClient.CurrentWorktreeAndBranch(ctx)
	if err != nil {
		return task.Git{}, fmt.Errorf("current worktree and branch: %w", err)
	}
	return task.Git{Worktree: worktree, Branch: branch}, nil
}
