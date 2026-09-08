// Package cli is the taskman CLI. It is a one-shot process: parse flags,
// run exactly one command, print, exit. There is no persistent daemon.
package cli

import (
	"github.com/urfave/cli/v3"

	taskcmd "github.com/mhmdkzr/taskman/internal/cli/task"
)

func rootCommand() *cli.Command {
	return &cli.Command{
		Name:  "taskman",
		Usage: "a file-backed, stateless, one-shot CLI task server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "git-dir", Value: ".", Usage: "repository root taskman operates against"},
			&cli.StringFlag{Name: "tasks-dir", Value: ".tasks", Usage: "directory holding task files"},
			&cli.StringFlag{
				Name:  "worktrees-dir",
				Value: ".worktrees",
				Usage: "directory task create creates worktrees under",
			},
			&cli.BoolFlag{Name: "json", Usage: "print the full JSON envelope instead of a human-readable summary"},
			&cli.StringFlag{Name: "log-level", Value: "info", Usage: "debug, info, warn, or error"},
			&cli.StringFlag{Name: "log-format", Value: "text", Usage: "text or json"},
		},
		Before:   initLogger,
		Commands: []*cli.Command{taskcmd.Command()},
	}
}
