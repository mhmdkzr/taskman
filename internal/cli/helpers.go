package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/prompts"
	"github.com/mhmdkzr/loop/internal/task"
)

func repoFrom(cmd *cli.Command) *task.Repo {
	return task.NewRepo(cmd.String("tasks-dir"))
}

func gitFrom(cmd *cli.Command) *task.GitClient {
	return task.NewGit(cmd.String("git-dir"))
}

func worktreesDirFrom(cmd *cli.Command) string {
	return cmd.String("worktrees-dir")
}

// requireID reads the task id from the command's first positional argument.
func requireID(cmd *cli.Command) (string, error) {
	id := cmd.Args().First()
	if id == "" {
		return "", cli.Exit("a task id is required", 2)
	}
	return id, nil
}

// splitKV parses "key=value" pairs (e.g. --label priority=high). value may
// itself contain "=" - only the first separator counts.
func splitKV(pairs []string) (map[string]string, error) {
	out := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			return nil, fmt.Errorf("malformed key=value pair: %q", pair)
		}
		out[key] = value
	}
	return out, nil
}

func parseChecks(pairs []string) (map[string]task.CheckResult, error) {
	kv, err := splitKV(pairs)
	if err != nil {
		return nil, err
	}
	checks := make(map[string]task.CheckResult, len(kv))
	for name, value := range kv {
		switch task.CheckResult(value) {
		case task.CheckOK, task.CheckError:
			checks[name] = task.CheckResult(value)
		default:
			return nil, fmt.Errorf("--check %s=%s: value must be %q or %q", name, value, task.CheckOK, task.CheckError)
		}
	}
	return checks, nil
}

func parseFindings(pairs []string) ([]task.Finding, error) {
	kv, err := splitKV(pairs)
	if err != nil {
		return nil, err
	}
	findings := make([]task.Finding, 0, len(kv))
	for file, detail := range kv {
		findings = append(findings, task.Finding{File: file, Detail: detail})
	}
	return findings, nil
}

// printTask writes t to stdout - the full JSON envelope behind --json, a
// short human-readable summary otherwise (design.md §7's "Shared request/
// response shapes").
func printTask(cmd *cli.Command, t task.Task) error {
	if cmd.Bool("json") {
		return printJSON(cmd, t)
	}
	if _, err := fmt.Fprintln(cmd.Root().Writer, prompts.TaskSummary{
		TaskID: t.ID,
		Title:  t.Title,
		State:  string(t.State),
		Stage:  currentStage(t),
	}.Render()); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// createSummary is task create's default (non-JSON) CLI output.
func createSummary(t task.Task) string {
	return prompts.CreateSummary{
		TaskID:   t.ID,
		Title:    t.Title,
		Worktree: t.Git.Worktree,
		Branch:   t.Git.Branch,
	}.Render()
}

// printGuidance writes g (task next's response) to stdout - the full JSON
// envelope behind --json, just g.Message otherwise.
func printGuidance(cmd *cli.Command, g task.Guidance) error {
	if cmd.Bool("json") {
		return printJSON(cmd, g)
	}
	if _, err := fmt.Fprintln(cmd.Root().Writer, g.Message); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func printJSON(cmd *cli.Command, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.Root().Writer, string(data)); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// currentStage describes, in one short phrase, which stage a task is
// waiting on - task_summary.md's Stage param.
func currentStage(t task.Task) string {
	switch t.State { //nolint:exhaustive // created/started fall through to the per-stage switch below
	case task.StateCompleted:
		return "merged"
	case task.StateFailed:
		return "abandoned"
	case task.StateBlocked:
		if t.Blocked != nil {
			return "blocked in " + t.Blocked.Stage
		}
		return "blocked"
	}
	switch {
	case t.Status.Specification.State != task.StageDone:
		return "awaiting specification"
	case t.Status.Implementation.State != task.StageDone:
		return "awaiting implementation"
	case t.Status.Verification.State != task.StageDone:
		return fmt.Sprintf("in verification (attempt %d)", max(t.Status.Verification.Attempts, 1))
	case task.NeedsFreshCommit(t):
		return "awaiting commit"
	case t.Status.Review.State == task.StageInProgress:
		return "in review-reject recovery"
	case t.Status.Review.State != task.StageDone:
		return "awaiting human review"
	case t.Status.Merge.State != task.StageDone:
		return "awaiting merge"
	default:
		return "awaiting completion"
	}
}

// exitCode maps taskman's error taxonomy to a process exit code -
// design.md §7's "Errors, one taxonomy" table.
func exitCode(err error) int {
	if exitErr, ok := errors.AsType[cli.ExitCoder](err); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func fail(err error) error {
	switch {
	case errors.Is(err, task.ErrTaskNotFound):
		return cli.Exit(err, 1)
	case errors.Is(err, task.ErrWorkingTreeDirty):
		return cli.Exit(err, 1)
	case errors.Is(err, task.ErrInvalidLabel):
		return cli.Exit(err, 2)
	default:
		if _, ok := errors.AsType[*task.InvalidTransitionError](err); ok {
			return cli.Exit(err, 1)
		}
		return cli.Exit(err, 1)
	}
}
