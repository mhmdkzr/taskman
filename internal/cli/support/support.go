// Package support holds the plumbing shared by every taskman CLI command:
// building a Repo/GitClient from the root flags, parsing repeated flag
// values, rendering output, and mapping errors to exit codes. It has no
// dependency on internal/cli or its subpackages, so any of them can import
// it without a cycle.
package support

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/prompts"
	"github.com/mhmdkzr/loop/internal/task"
)

// RepoFrom builds a task.Repo rooted at the --tasks-dir root flag.
func RepoFrom(cmd *cli.Command) *task.Repo {
	return task.NewRepo(cmd.String("tasks-dir"))
}

// GitFrom builds a task.GitClient rooted at the --git-dir root flag.
func GitFrom(cmd *cli.Command) *task.GitClient {
	return task.NewGit(cmd.String("git-dir"))
}

// WorktreesDirFrom reads the --worktrees-dir root flag.
func WorktreesDirFrom(cmd *cli.Command) string {
	return cmd.String("worktrees-dir")
}

// RequireID reads the task id from the command's first positional argument.
func RequireID(cmd *cli.Command) (string, error) {
	id := cmd.Args().First()
	if id == "" {
		return "", cli.Exit("a task id is required", 2)
	}
	return id, nil
}

// SplitKV parses "key=value" pairs (e.g. --label priority=high). value may
// itself contain "=" - only the first separator counts.
func SplitKV(pairs []string) (map[string]string, error) {
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

// ParseChecks parses repeated --check name=ok|error pairs.
func ParseChecks(pairs []string) (map[string]task.CheckResult, error) {
	kv, err := SplitKV(pairs)
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

// ParseFindings parses repeated --finding file=detail pairs.
func ParseFindings(pairs []string) ([]task.Finding, error) {
	kv, err := SplitKV(pairs)
	if err != nil {
		return nil, err
	}
	findings := make([]task.Finding, 0, len(kv))
	for file, detail := range kv {
		findings = append(findings, task.Finding{File: file, Detail: detail})
	}
	return findings, nil
}

// PrintTask writes t to stdout - the full JSON envelope behind --json, a
// short human-readable summary otherwise.
func PrintTask(cmd *cli.Command, t task.Task) error {
	if cmd.Bool("json") {
		return PrintJSON(cmd, t)
	}
	if _, err := fmt.Fprintln(cmd.Root().Writer, prompts.TaskSummary{
		TaskID: t.ID,
		Title:  t.Title,
		State:  string(t.State),
		Stage:  CurrentStage(t),
	}.Render()); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// CreateSummary is task create's default (non-JSON) CLI output.
func CreateSummary(t task.Task) string {
	return prompts.CreateSummary{
		TaskID:   t.ID,
		Title:    t.Title,
		Worktree: t.Git.Worktree,
		Branch:   t.Git.Branch,
	}.Render()
}

// PrintGuidance writes g (task next's response) to stdout - the full JSON
// envelope behind --json, just g.Message otherwise.
func PrintGuidance(cmd *cli.Command, g task.Guidance) error {
	if cmd.Bool("json") {
		return PrintJSON(cmd, g)
	}
	if _, err := fmt.Fprintln(cmd.Root().Writer, g.Message); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// PrintJSON writes v to stdout as indented JSON.
func PrintJSON(cmd *cli.Command, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.Root().Writer, string(data)); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// CurrentStage describes, in one short phrase, which stage a task is
// waiting on - task_summary.md's Stage param.
func CurrentStage(t task.Task) string {
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

// ExitCode maps a returned error to a process exit code.
func ExitCode(err error) int {
	if exitErr, ok := errors.AsType[cli.ExitCoder](err); ok {
		return exitErr.ExitCode()
	}
	return 1
}

// Fail wraps a domain error from internal/task into a cli.ExitCoder with
// the right exit code.
func Fail(err error) error {
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
