// Package cli is the taskman CLI. It is a one-shot process: parse flags,
// run exactly one command, print, exit. There is no persistent daemon.
// Every command is mounted directly on the root - there is no intermediate
// "task" grouping command, except for the two review-gate groupings
// (specification, implementation), each of which mounts its own
// review/{agent,human}/{approved,rejected} tree.
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/abandoned"
	"github.com/mhmdkzr/taskman/internal/commands/committed"
	"github.com/mhmdkzr/taskman/internal/commands/create"
	"github.com/mhmdkzr/taskman/internal/commands/escalated"
	"github.com/mhmdkzr/taskman/internal/commands/get"
	"github.com/mhmdkzr/taskman/internal/commands/implementation"
	"github.com/mhmdkzr/taskman/internal/commands/implemented"
	"github.com/mhmdkzr/taskman/internal/commands/list"
	"github.com/mhmdkzr/taskman/internal/commands/mcp"
	"github.com/mhmdkzr/taskman/internal/commands/merged"
	"github.com/mhmdkzr/taskman/internal/commands/skill"
	"github.com/mhmdkzr/taskman/internal/commands/specification"
	"github.com/mhmdkzr/taskman/internal/commands/specified"
	"github.com/mhmdkzr/taskman/internal/commands/verified"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Run parses os.Args, runs exactly one command, and returns the process
// exit code.
func Run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := rootCommand().Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return utils.ExitCode(err)
	}
	return 0
}

func rootCommand() *cli.Command {
	return &cli.Command{
		Name:  "taskman",
		Usage: "a SQLite-backed, event-sourced, one-shot CLI task server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "git-dir", Value: ".", Usage: "repository root taskman reads commits from"},
			&cli.StringFlag{Name: "db", Value: "./tasks.db", Usage: "path to the SQLite task database"},
			&cli.BoolFlag{Name: "json", Usage: "print the full JSON envelope instead of a human-readable summary"},
			&cli.BoolFlag{
				Name:  "md",
				Usage: "render the task as a Markdown document instead of a human-readable summary",
			},
			&cli.StringFlag{Name: "log-level", Value: "info", Usage: "debug, info, warn, or error"},
			&cli.StringFlag{Name: "log-format", Value: "text", Usage: "text or json"},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// --json and --md are two renderings of the same output; asking
			// for both is malformed input, not a silent precedence.
			if cmd.Bool("json") && cmd.Bool("md") {
				return ctx, cli.Exit("--json and --md are mutually exclusive", 2)
			}
			return initLogger(ctx, cmd)
		},
		Commands: commands(),
	}
}

func commands() []*cli.Command {
	return []*cli.Command{
		create.Command(),
		specified.Command(),
		specification.Command(),
		implemented.Command(),
		implementation.Command(),
		verified.Command(),
		committed.Command(),
		merged.Command(),
		escalated.Command(),
		abandoned.Command(),
		get.Command(),
		list.Command(),
		skill.Command(),
		mcp.Command(),
	}
}

// initLogger sets up slog per the root --log-level/--log-format flags,
// before any command runs.
func initLogger(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cmd.String("log-level"))); err != nil {
		return ctx, fmt.Errorf("parse --log-level: %w", err)
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level, AddSource: true}
	switch format := cmd.String("log-format"); format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		return ctx, fmt.Errorf("--log-format must be %q or %q, got %q", "text", "json", format)
	}
	slog.SetDefault(slog.New(handler))
	return ctx, nil
}
