// Package codebase wraps internal/codebase's Repository operations as
// goai.Tool values, bound to one Repository (one task's worktree) at
// construction time — unlike the process-wide tool sets this replaces, a
// fresh tool set is built per task run, so an agent's file access is scoped
// to exactly the codebase it was given.
package codebase

import (
	cb "github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/zendev-sh/goai"
)

// Tools returns the full read+write tool set (read, edit, glob, grep) bound
// to repo — the executor's tool set.
func Tools(repo cb.Repository) []goai.Tool {
	return append(ReadOnlyTools(repo), EditTool(repo))
}

// ReadOnlyTools returns read, glob, grep bound to repo, with no way to
// mutate the worktree — the reviewer's tool set. A reviewer built from this
// set can find and report problems but cannot "helpfully" fix them itself,
// which is enforced here structurally rather than by prompting.
func ReadOnlyTools(repo cb.Repository) []goai.Tool {
	return []goai.Tool{ReadTool(repo), GlobTool(repo), GrepTool(repo)}
}
