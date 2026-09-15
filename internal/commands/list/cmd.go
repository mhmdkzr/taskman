package list

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	jsonview "github.com/mhmdkzr/taskman/internal/task/view/json"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// defaultLimit caps a call's page size when --limit isn't given, so `list`
// against a large store doesn't dump everything at once.
const defaultLimit = 50

// Command returns the "list" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list tasks, optionally filtered and paginated",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "state", Usage: "filter by task state - repeatable"},
			&cli.StringSliceFlag{Name: "label", Usage: "filter by label as key=value - repeatable"},
			&cli.IntFlag{Name: "limit", Value: defaultLimit, Usage: "max tasks to return; 0 for unlimited"},
			&cli.IntFlag{Name: "offset", Usage: "skip this many matching tasks before the page starts"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			req, err := requestFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			st, err := store.Open(ctx, cmd.String("db"))
			if err != nil {
				return utils.Fail(err)
			}
			defer func() {
				if err := st.Close(); err != nil {
					slog.Error("close store", "error", err)
				}
			}()

			tasks, total, err := List(ctx, st, req)
			if err != nil {
				return utils.Fail(err)
			}
			if cmd.Bool("json") {
				docs := make([]jsonview.Document, 0, len(tasks))
				for _, t := range tasks {
					docs = append(docs, jsonview.FromTask(t))
				}
				return view.PrintJSON(cmd, Result{
					Tasks:  docs,
					Total:  total,
					Limit:  req.Limit,
					Offset: req.Offset,
				})
			}
			if len(tasks) == 0 {
				_, err := fmt.Fprintln(cmd.Root().Writer, "no tasks")
				return utils.Fail(err)
			}
			for i, t := range tasks {
				if cmd.Bool("md") {
					if i > 0 {
						if _, err := fmt.Fprintln(cmd.Root().Writer, "\n---"); err != nil {
							return utils.Fail(err)
						}
					}
					if err := view.PrintMarkdown(cmd, t); err != nil {
						return utils.Fail(err)
					}
					continue
				}
				line := fmt.Sprintf("%s - %s - [%s]", t.ID, t.Definition.Title, t.State())
				if _, err := fmt.Fprintln(cmd.Root().Writer, line); err != nil {
					return utils.Fail(err)
				}
			}
			if shown := req.Offset + len(tasks); shown < total {
				if _, err := fmt.Fprintf(cmd.Root().Writer,
					"... %d more (use --offset %d to see the rest)\n", total-shown, shown,
				); err != nil {
					return utils.Fail(err)
				}
			}
			return nil
		},
	}
}

// requestFrom builds list's Request from its flags, rejecting an unknown
// --state or a malformed --label as malformed input.
func requestFrom(cmd *cli.Command) (Request, error) {
	rawStates := cmd.StringSlice("state")
	states := make([]task.TaskState, len(rawStates))
	for i, raw := range rawStates {
		state, err := task.ParseTaskState(raw)
		if err != nil {
			return Request{}, cli.Exit(fmt.Sprintf("--state: %v", err), 2)
		}
		states[i] = state
	}
	labels, err := utils.SplitKV(cmd.StringSlice("label"))
	if err != nil {
		return Request{}, cli.Exit(fmt.Sprintf("--label: %v", err), 2)
	}
	return Request{
		States: states,
		Labels: labels,
		Limit:  cmd.Int("limit"),
		Offset: cmd.Int("offset"),
	}, nil
}
