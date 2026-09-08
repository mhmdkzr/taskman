package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/task"
)

//nolint:dupl // thin per-command CLI wiring, per design.md §7 - shares a body shape with escalateCommand but a distinct flag surface
func specifyCommand() *cli.Command {
	return &cli.Command{
		Name:  "specify",
		Usage: "write a task's specification and done_when",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "result", Required: true},
			&cli.StringFlag{Name: "done-when", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			t, err := task.Specify(repoFrom(cmd), id, task.SpecifyRequest{
				Result:   cmd.String("result"),
				DoneWhen: cmd.String("done-when"),
			})
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

func implementCommand() *cli.Command {
	return &cli.Command{
		Name:  "implement",
		Usage: "mark a task's implementation attempt as done",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			t, err := task.Implement(repoFrom(cmd), id)
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

func verifyCommand() *cli.Command {
	return &cli.Command{
		Name:  "verify",
		Usage: "report one build-check attempt",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "check", Usage: "name=ok|error (repeatable)", Required: true},
			&cli.StringFlag{Name: "output"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			checks, err := parseChecks(cmd.StringSlice("check"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			t, err := task.Verify(repoFrom(cmd), id, task.VerifyRequest{
				Checks: checks,
				Output: cmd.String("output"),
			})
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

func reviewRecordCommand() *cli.Command {
	return &cli.Command{
		Name:  "record",
		Usage: "report the automated review round's verdict",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "approved", Required: true},
			&cli.StringSliceFlag{Name: "finding", Usage: "file=detail (repeatable)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			findings, err := parseFindings(cmd.StringSlice("finding"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			t, err := task.RecordReview(repoFrom(cmd), id, task.ReviewRecordRequest{
				Approved: cmd.Bool("approved"),
				Findings: findings,
			})
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

func reviewApproveCommand() *cli.Command {
	return &cli.Command{
		Name:  "approve",
		Usage: "record a human's approval at the review stage",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "comment"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			t, err := task.ApproveReview(repoFrom(cmd), id, cmd.String("comment"))
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

//nolint:dupl // thin per-command CLI wiring, per design.md §7 - shares a body shape with abandonCommand but a distinct flag surface
func reviewRejectCommand() *cli.Command {
	return &cli.Command{
		Name:  "reject",
		Usage: "record a human's rejection and start review-reject recovery",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "reason", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			t, err := task.RejectReview(repoFrom(cmd), id, cmd.String("reason"))
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

func commitCommand() *cli.Command {
	return &cli.Command{
		Name:  "commit",
		Usage: "read back the commit the caller already made",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "commit", Usage: "commit-ish to read instead of HEAD"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			t, err := task.Commit(ctx, repoFrom(cmd), gitFrom(cmd), id, task.CommitRequest{
				Commit: cmd.String("commit"),
			})
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

//nolint:dupl // thin per-command CLI wiring, per design.md §7 - shares a body shape with specifyCommand but a distinct flag surface
func escalateCommand() *cli.Command {
	return &cli.Command{
		Name:  "escalate",
		Usage: "block a task on a dispatched agent's own escalate call",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "stage", Required: true},
			&cli.StringFlag{Name: "reason", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			t, err := task.Escalate(repoFrom(cmd), id, task.EscalateRequest{
				Stage:  cmd.String("stage"),
				Reason: cmd.String("reason"),
			})
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

func mergeCommand() *cli.Command {
	return &cli.Command{
		Name:  "merge",
		Usage: "record that the caller already merged the task's branch",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "commit", Usage: "override the recorded commit hash"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			t, err := task.Merge(repoFrom(cmd), id, task.MergeRequest{Commit: cmd.String("commit")})
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

//nolint:dupl // thin per-command CLI wiring, per design.md §7 - shares a body shape with reviewRejectCommand but a distinct flag surface
func abandonCommand() *cli.Command {
	return &cli.Command{
		Name:  "abandon",
		Usage: "mark a task failed for good",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "reason", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			t, err := task.Abandon(repoFrom(cmd), id, cmd.String("reason"))
			if err != nil {
				return fail(err)
			}
			return printTask(cmd, t)
		},
	}
}

func nextCommand() *cli.Command {
	return &cli.Command{
		Name:  "next",
		Usage: "what should happen next for this task",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := requireID(cmd)
			if err != nil {
				return err
			}
			g, err := task.Next(repoFrom(cmd), id)
			if err != nil {
				return fail(err)
			}
			return printGuidance(cmd, g)
		},
	}
}
