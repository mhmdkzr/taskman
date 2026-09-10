// Package utils holds the plumbing shared by every taskman CLI command's
// cmd.go: building a GitClient from the root flags, parsing repeated flag
// values, rendering output, and mapping errors to exit codes. It has no
// dependency on internal/commands or any slice under it, so any of them can
// import it without a cycle.
package utils

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
)

//go:embed task_summary.md
var taskSummaryFile embed.FS

var taskSummaryTmpl = template.Must(template.New("task_summary.md").
	ParseFS(taskSummaryFile, "task_summary.md"))

// taskSummary is the default (non-JSON) CLI output for any command that
// returns a Task rather than a next.Guidance - the one prompt template
// genuinely shared by many command slices, so it lives here rather than in
// any one of them.
type taskSummary struct {
	TaskID string
	Title  string
	State  string
	Stage  string
}

func (p taskSummary) render() string {
	var b strings.Builder
	if err := taskSummaryTmpl.Execute(&b, p); err != nil {
		panic(fmt.Sprintf("support: render task_summary: %v", err))
	}
	return strings.TrimRight(b.String(), "\n")
}

// GitFrom builds a Git client rooted at the --git-dir root flag.
func GitFrom(cmd *cli.Command) *git.Client {
	return git.NewClient(cmd.String("git-dir"))
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
	if _, err := fmt.Fprintln(cmd.Root().Writer, taskSummary{
		TaskID: t.ID,
		Title:  t.Title,
		State:  string(t.State),
		Stage:  CurrentStage(t),
	}.render()); err != nil {
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
	switch t.State {
	case task.StateSpecify:
		return "awaiting specification"
	case task.StateSpecificationReview:
		return "awaiting specification approval"
	case task.StateImplement:
		return "awaiting implementation"
	case task.StateVerify:
		return "awaiting verification"
	case task.StateFixVerificationFailure:
		return "fixing verification failure"
	case task.StateFixAutomatedReviewFindings:
		return "fixing automated review findings"
	case task.StateAutomatedReview:
		return fmt.Sprintf("awaiting automated review (attempt %d)", len(t.Reviews)+1)
	case task.StateCommit:
		return "awaiting commit"
	case task.StateHumanReview:
		return "awaiting human review"
	case task.StateFixHumanReviewFindings:
		return "fixing human review findings"
	case task.StateMerge:
		return "awaiting merge"
	case task.StateCompleted:
		return "completed"
	case task.StateAbandoned:
		return "abandoned"
	case task.StateBlocked:
		if t.Blocked != nil {
			return "blocked in " + t.Blocked.Stage
		}
		return "blocked"
	default:
		return "unknown"
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
