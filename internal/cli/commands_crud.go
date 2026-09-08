package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/task"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list tasks, optionally filtered",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "state", Usage: "filter by task.State (repeatable)"},
			&cli.StringSliceFlag{Name: "label", Usage: "filter by label k=v (repeatable)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			stateFlags := cmd.StringSlice("state")
			states := make([]task.State, len(stateFlags))
			for i, s := range stateFlags {
				states[i] = task.State(s)
			}
			labels, err := splitKV(cmd.StringSlice("label"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			tasks, err := repoFrom(cmd).List(task.Filter{State: states, Labels: labels})
			if err != nil {
				return fail(err)
			}
			if cmd.Bool("json") {
				return printJSON(cmd, tasks)
			}
			if len(tasks) == 0 {
				if _, err := fmt.Fprintln(cmd.Root().Writer, "no tasks"); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
				return nil
			}
			for _, t := range tasks {
				if err := printTask(cmd, t); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func getCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "show one task",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			t, err := repoFrom(cmd).Get(id)
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

func createCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "create a task and its worktree/branch",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "definition", Required: true},
			&cli.StringFlag{Name: "id"},
			&cli.StringFlag{Name: "title"},
			&cli.StringSliceFlag{Name: "label", Usage: "k=v (repeatable)"},
			&cli.StringSliceFlag{Name: "reference", Usage: "repeatable"},
			&cli.StringFlag{Name: "specification"},
			&cli.StringFlag{Name: "done-when"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			labels, err := splitKV(cmd.StringSlice("label"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			t, err := task.Create(ctx, repoFrom(cmd), gitFrom(cmd), worktreesDirFrom(cmd), task.CreateRequest{
				Definition:    cmd.String("definition"),
				ID:            cmd.String("id"),
				Title:         cmd.String("title"),
				Labels:        labels,
				References:    cmd.StringSlice("reference"),
				Specification: cmd.String("specification"),
				DoneWhen:      cmd.String("done-when"),
			})
			if err != nil {
				return fail(err)
			}
			if cmd.Bool("json") {
				return printJSON(cmd, t)
			}
			if _, err := fmt.Fprintln(cmd.Root().Writer, createSummary(t)); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			return nil
		},
	}
}

func updateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "patch a task's title, labels, or references",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "title"},
			&cli.StringSliceFlag{Name: "label", Usage: "k=v (repeatable)"},
			&cli.StringSliceFlag{Name: "unset-label", Usage: "repeatable"},
			&cli.StringSliceFlag{Name: "reference", Usage: "repeatable"},
			&cli.BoolFlag{Name: "clear-references"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			labels, err := splitKV(cmd.StringSlice("label"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			req := task.UpdateRequest{
				SetLabels:       labels,
				UnsetLabels:     cmd.StringSlice("unset-label"),
				References:      cmd.StringSlice("reference"),
				ClearReferences: cmd.Bool("clear-references"),
			}
			if cmd.IsSet("title") {
				title := cmd.String("title")
				req.Title = &title
			}
			t, err := task.Update(repoFrom(cmd), id, req)
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

func deleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "remove a task file outright - a human housekeeping action",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			if err := task.Delete(repoFrom(cmd), id); err != nil {
				return fail(err)
			}
			if _, err := fmt.Fprintf(cmd.Root().Writer, "deleted task %s\n", id); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			return nil
		},
	}
}
