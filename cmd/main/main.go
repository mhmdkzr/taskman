// Command taskman is the CLI entrypoint for taskman - see
// notes/design/design.md §7. It is a one-shot process: parse flags, run
// exactly one command, print, exit. There is no persistent daemon.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v3"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := rootCommand().Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitCode(err)
	}
	return 0
}

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
		Commands: []*cli.Command{taskCommand()},
	}
}

func taskCommand() *cli.Command {
	return &cli.Command{
		Name:  "task",
		Usage: "manage tasks - see notes/design/design.md §6",
		Commands: []*cli.Command{
			listCommand(),
			getCommand(),
			createCommand(),
			updateCommand(),
			specifyCommand(),
			implementCommand(),
			verifyCommand(),
			reviewCommand(),
			commitCommand(),
			escalateCommand(),
			mergeCommand(),
			abandonCommand(),
			nextCommand(),
			deleteCommand(),
		},
	}
}

func reviewCommand() *cli.Command {
	return &cli.Command{
		Name:  "review",
		Usage: "the review stage - automated (record) and human (approve/reject)",
		Commands: []*cli.Command{
			reviewRecordCommand(),
			reviewApproveCommand(),
			reviewRejectCommand(),
		},
	}
}
