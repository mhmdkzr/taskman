// Package codebase wraps internal/codebase's Repository operations as
// goai.Tool values, bound to one Repository (one task's worktree) at
// construction time — unlike the process-wide tool sets this replaces, a
// fresh tool set is built per task run, so an agent's file access is scoped
// to exactly the codebase it was given.
package codebase

import (
	"github.com/zendev-sh/goai"

	cb "github.com/mhmdkzr/taskman/internal/codebase"
)

// Tools returns the full read+write tool set (read, edit, glob, grep, plus
// the compile/test feedback loop) bound to repo — the executor's tool set.
//
// gofmt, goimports, go vet, staticcheck, and golangci-lint are deliberately
// NOT tools: they're deterministic and don't need agent judgment to run, so
// the pipeline runs them itself after the agent is done, rather than leaving
// it up to the agent to remember to call them (see
// internal/codebase.Repository.Format/Lint).
func Tools(repo cb.Repository) []goai.Tool {
	return append(ReadOnlyTools(repo), EditTool(repo))
}

// ReadOnlyTools returns read, glob, grep, go_build, go_test bound to repo,
// with no way to mutate the worktree — the reviewer's tool set. A reviewer
// built from this set can find and report problems (including by running the
// build/tests itself) but cannot "helpfully" fix them, which is enforced
// here structurally rather than by prompting.
func ReadOnlyTools(repo cb.Repository) []goai.Tool {
	return []goai.Tool{
		ReadTool(repo), GlobTool(repo), GrepTool(repo),
		GoBuildTool(repo), GoTestTool(repo),
	}
}
