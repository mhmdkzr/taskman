// Package utils holds the plumbing shared by every taskman CLI command's
// cmd.go: building a Git client and a store handle from the root flags,
// parsing repeated flag values, rendering output, and mapping errors to
// exit codes. It has no dependency on internal/commands or any slice under
// it, so any of them can import it without a cycle.
package utils

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/urfave/cli/v3"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

//go:embed task_summary.md
var taskSummaryFile embed.FS

var taskSummaryTmpl = template.Must(template.New("task_summary.md").
	ParseFS(taskSummaryFile, "task_summary.md"))

// taskSummary is the default (non-JSON) CLI output for any command that
// returns a Task - the one prompt template genuinely shared by every
// command slice, so it lives here rather than in any one of them.
type taskSummary struct {
	TaskID      string
	State       string
	Instruction string
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

// StoreFrom opens the task store at the --db root flag. Callers are
// responsible for closing it.
func StoreFrom(cmd *cli.Command) (*store.Store, error) {
	st, err := store.Open(cmd.String("db"))
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	return st, nil
}

// IDFrom parses the --id flag as a task id.
func IDFrom(cmd *cli.Command) (uuid.UUID, error) {
	raw := cmd.String("id")
	if raw == "" {
		return uuid.Nil(), cli.Exit("--id is required", 2)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil(), cli.Exit(fmt.Sprintf("--id: %v", err), 2)
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

// ParseFindings parses repeated --finding location=detail pairs.
func ParseFindings(pairs []string) ([]task.Finding, error) {
	kv, err := SplitKV(pairs)
	if err != nil {
		return nil, err
	}
	findings := make([]task.Finding, 0, len(kv))
	for location, detail := range kv {
		findings = append(findings, task.Finding{Location: location, Detail: detail})
	}
	return findings, nil
}

// ParseCheckResult parses a --unit/--integration/--end-to-end/--linters
// flag's value. An empty value means the check wasn't reported.
func ParseCheckResult(flag, value string) (task.CheckResult, error) {
	switch task.CheckResult(value) {
	case "":
		return "", nil
	case task.CheckOK, task.CheckError:
		return task.CheckResult(value), nil
	default:
		return "", cli.Exit(fmt.Sprintf("--%s: value must be %q or %q, got %q", flag, task.CheckOK, task.CheckError, value), 2)
	}
}

// ReviewFlags are shared by every command that reports a review gate's
// configuration (specified, implemented): whether an agent/human review is
// required, and each one's auto-fix policy.
func ReviewFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "agent-review", Usage: "require an automated review"},
		&cli.BoolFlag{Name: "agent-review-use-subagent", Usage: "run the automated review in a subagent"},
		&cli.BoolFlag{Name: "agent-review-auto-fix", Usage: "automatically fix automated review findings"},
		&cli.IntFlag{Name: "agent-review-auto-fix-max-rounds", Usage: "max automated-review auto-fix rounds"},
		&cli.BoolFlag{Name: "agent-review-auto-fix-use-subagent", Usage: "run automated-review auto-fix in a subagent"},
		&cli.BoolFlag{Name: "human-review", Usage: "require a human review"},
		&cli.BoolFlag{Name: "human-review-auto-fix", Usage: "automatically fix human review findings"},
		&cli.IntFlag{Name: "human-review-auto-fix-max-rounds", Usage: "max human-review auto-fix rounds"},
		&cli.BoolFlag{Name: "human-review-auto-fix-use-subagent", Usage: "run human-review auto-fix in a subagent"},
	}
}

// ReviewConfigurationFrom builds a task.ReviewConfiguration from the flags
// ReviewFlags declares.
func ReviewConfigurationFrom(cmd *cli.Command) task.ReviewConfiguration {
	return task.ReviewConfiguration{
		Agent: task.AgentReviewConfiguration{
			Required:    cmd.Bool("agent-review"),
			UseSubagent: cmd.Bool("agent-review-use-subagent"),
			AutoFix: task.AutoFix{
				Enabled:     cmd.Bool("agent-review-auto-fix"),
				MaxRounds:   cmd.Int("agent-review-auto-fix-max-rounds"),
				UseSubagent: cmd.Bool("agent-review-auto-fix-use-subagent"),
			},
		},
		Human: task.HumanReviewConfiguration{
			Required: cmd.Bool("human-review"),
			AutoFix: task.AutoFix{
				Enabled:     cmd.Bool("human-review-auto-fix"),
				MaxRounds:   cmd.Int("human-review-auto-fix-max-rounds"),
				UseSubagent: cmd.Bool("human-review-auto-fix-use-subagent"),
			},
		},
	}
}

// PrintTask writes t to stdout - the full JSON envelope behind --json, a
// short human-readable summary otherwise.
func PrintTask(cmd *cli.Command, t task.Task) error {
	if cmd.Bool("json") {
		return PrintJSON(cmd, t)
	}
	instruction := t.Instruction()
	if _, err := fmt.Fprintln(cmd.Root().Writer, taskSummary{
		TaskID:      t.ID.String(),
		State:       t.State().String(),
		Instruction: fmt.Sprintf("%s (%s)", instruction.Action, instruction.State),
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

// ExitCode maps a returned error to a process exit code.
func ExitCode(err error) int {
	if exitErr, ok := errors.AsType[cli.ExitCoder](err); ok {
		return exitErr.ExitCode()
	}
	return 1
}

// Fail wraps a domain error into a cli.ExitCoder. Flag-parsing/usage errors
// are exit code 2 and are returned directly via cli.Exit at the call site;
// every domain error reaching here is a runtime failure, exit code 1.
func Fail(err error) error {
	return cli.Exit(err, 1)
}
