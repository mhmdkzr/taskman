// Package task wires up every `taskman task <command>` subcommand: parsing
// flags into a request, calling straight into internal/task's matching
// function, and rendering the result via internal/cli/support.
package task

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/task/abandon"
	"github.com/mhmdkzr/taskman/internal/cli/task/commit"
	"github.com/mhmdkzr/taskman/internal/cli/task/create"
	"github.com/mhmdkzr/taskman/internal/cli/task/delete"
	"github.com/mhmdkzr/taskman/internal/cli/task/escalate"
	"github.com/mhmdkzr/taskman/internal/cli/task/get"
	"github.com/mhmdkzr/taskman/internal/cli/task/implement"
	"github.com/mhmdkzr/taskman/internal/cli/task/list"
	"github.com/mhmdkzr/taskman/internal/cli/task/merge"
	"github.com/mhmdkzr/taskman/internal/cli/task/next"
	"github.com/mhmdkzr/taskman/internal/cli/task/review"
	"github.com/mhmdkzr/taskman/internal/cli/task/specify"
	"github.com/mhmdkzr/taskman/internal/cli/task/update"
	"github.com/mhmdkzr/taskman/internal/cli/task/verify"
)

// Command builds the "task" command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "task",
		Usage: "create and drive tasks through their lifecycle",
		Commands: []*cli.Command{
			list.Command(),
			get.Command(),
			create.Command(),
			update.Command(),
			specify.Command(),
			implement.Command(),
			verify.Command(),
			review.Command(),
			commit.Command(),
			escalate.Command(),
			merge.Command(),
			abandon.Command(),
			next.Command(),
			delete.Command(),
		},
	}
}
