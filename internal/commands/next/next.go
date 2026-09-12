// Package next owns the "next" command: it reads a task and reports what
// should happen next as actionable guidance - a rendered message plus the
// command(s) that report the outcome - rather than the bare
// task.Instruction projection get already exposes.
package next

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"sort"
	"strings"
	tmpl "text/template"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

//go:embed guidance.md
var guidanceTemplate string

var tpl = tmpl.Must(tmpl.New("guidance.md").Parse(guidanceTemplate))

// Guidance is next's output: the workflow's current instruction for a task,
// rendered as a message for whichever caller (human or agent) is meant to
// act on it, plus the command(s) that would currently report an outcome.
type Guidance struct {
	TaskID   uuid.UUID              `json:"task_id"`
	State    task.TaskState         `json:"state"`
	Action   task.InstructionAction `json:"action"`
	Message  string                 `json:"message"`
	Commands []string               `json:"commands,omitempty"`
}

// Request is next's input.
type Request struct {
	ID uuid.UUID `json:"id" jsonschema:"the task id to inspect"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Next reports what should happen next for req's task.
func Next(ctx context.Context, st *store.Store, req Request) (Guidance, error) {
	if err := req.validate(); err != nil {
		return Guidance{}, fmt.Errorf("next: %w", err)
	}
	t, err := st.Read(ctx, req.ID)
	if err != nil {
		return Guidance{}, fmt.Errorf("next: %w", err)
	}
	instruction := t.Instruction()
	message, err := renderMessage(t, instruction)
	if err != nil {
		return Guidance{}, fmt.Errorf("next: %w", err)
	}
	return Guidance{
		TaskID:   t.ID,
		State:    instruction.State,
		Action:   instruction.Action,
		Message:  message,
		Commands: commandsFor(t),
	}, nil
}

// view is the data guidance.md's per-state templates render against.
type view struct {
	Task   task.Task
	Reason string
}

// LastCommitHash is the implementation's most recent recorded commit, or ""
// before one is recorded.
func (v view) LastCommitHash() string {
	if v.Task.Implementation == nil || len(v.Task.Implementation.Git.Commits) == 0 {
		return ""
	}
	commits := v.Task.Implementation.Git.Commits
	return commits[len(commits)-1].Hash
}

func renderMessage(t task.Task, instruction task.Instruction) (string, error) {
	var buf bytes.Buffer
	data := view{Task: t, Reason: reasonFor(t, instruction.State)}
	if err := tpl.ExecuteTemplate(&buf, string(instruction.State), data); err != nil {
		return "", fmt.Errorf("render guidance: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

// reasonFor explains why a "fix" state was reached - the failure or
// rejection its dispatched agent needs to address. Every other state's
// guidance is self-explanatory from the task alone.
func reasonFor(t task.Task, state task.TaskState) string {
	switch state {
	case task.StateFixVerificationFailure:
		return verificationFailureReason(t)
	case task.StateFixAutomatedReviewFindings:
		return automatedReviewFindingsReason(t)
	case task.StateFixHumanReviewFindings:
		return humanReviewFindingsReason(t)
	default:
		return ""
	}
}

func verificationFailureReason(t task.Task) string {
	attempts := t.Implementation.Verification.Attempts
	if len(attempts) == 0 {
		return "Verification failed."
	}
	last := attempts[len(attempts)-1]
	reason := "Verification failed"
	var failed []string
	if last.Checks.Unit == task.CheckError {
		failed = append(failed, "unit")
	}
	if last.Checks.Integration == task.CheckError {
		failed = append(failed, "integration")
	}
	if last.Checks.EndToEnd == task.CheckError {
		failed = append(failed, "end-to-end")
	}
	if last.Checks.Linters == task.CheckError {
		failed = append(failed, "linters")
	}
	if len(failed) > 0 {
		reason += " (" + strings.Join(failed, ", ") + ")"
	}
	if last.Output != "" {
		return reason + ":\n" + last.Output
	}
	return reason + "."
}

func automatedReviewFindingsReason(t task.Task) string {
	results := t.Implementation.Review.Agent.Results
	if len(results) == 0 {
		return "The automated reviewer reported findings that need to be fixed."
	}
	last := results[len(results)-1]
	var b strings.Builder
	b.WriteString("Automated review rejected this attempt:\n")
	for _, finding := range last.Findings {
		fmt.Fprintf(&b, "- %s: %s\n", finding.Location, finding.Detail)
	}
	return strings.TrimRight(b.String(), "\n")
}

func humanReviewFindingsReason(t task.Task) string {
	var lastVerification *task.VerificationResult
	if attempts := t.Implementation.Verification.Attempts; len(attempts) > 0 {
		lastVerification = &attempts[len(attempts)-1]
	}
	var lastHuman *task.HumanReviewResult
	if results := t.Implementation.Review.Human.Results; len(results) > 0 {
		lastHuman = &results[len(results)-1]
	}
	if lastVerification != nil && !lastVerification.Passed &&
		(lastHuman == nil || lastVerification.At.After(lastHuman.At)) {
		return verificationFailureReason(t)
	}
	if lastHuman != nil {
		return "A human rejected this task's review: " + lastHuman.Comment
	}
	return "The human-review changes need another verification pass."
}

// commandNames maps each EventKind to the CLI invocation that reports it,
// with %s standing in for the task id. It is the one place next needs to
// know each slice's command name - the rest of a command's flags aren't
// reconstructable from the task alone (a plan, check results, a commit
// target, ...), so placeholders stand in for those.
var commandNames = map[task.EventKind]string{
	task.EventSpecificationSubmitted: "specified --id %s --plan <plan text> [--agent-review] [--human-review]",
	task.EventSpecificationReviewAgentApproved: "specification review agent approved --id %s " +
		"[--comment <text>]",
	task.EventSpecificationReviewAgentRejected: "specification review agent rejected --id %s " +
		"--finding <location>=<detail> [...]",
	task.EventSpecificationReviewHumanApproved: "specification review human approved --id %s " +
		"[--comment <text>]",
	task.EventSpecificationReviewHumanRejected: "specification review human rejected --id %s --reason <text>",
	task.EventImplementationCompleted: "implemented --id %s --worktree <path> --branch <name> " +
		"[--unit] [--integration] [--end-to-end] [--linters] [--agent-review] [--human-review]",
	task.EventVerificationPassed: "verified --id %s --unit <ok|error> --integration <ok|error> " +
		"--end-to-end <ok|error> --linters <ok|error> [--output <text>]",
	task.EventVerificationFailed: "verified --id %s --unit <ok|error> --integration <ok|error> " +
		"--end-to-end <ok|error> --linters <ok|error> [--output <text>]",
	task.EventImplementationReviewAgentApproved: "implementation review agent approved --id %s " +
		"[--comment <text>]",
	task.EventImplementationReviewAgentRejected: "implementation review agent rejected --id %s " +
		"--finding <location>=<detail> [...]",
	task.EventCommitRecorded: "committed --id %s",
	task.EventImplementationReviewHumanApproved: "implementation review human approved --id %s " +
		"[--comment <text>]",
	task.EventImplementationReviewHumanRejected: "implementation review human rejected --id %s --reason <text>",
	task.EventMergeCompleted:                    "merged --id %s --target <branch>",
}

// commandsFor lists every command that would currently be accepted for t,
// derived from task.ValidEvents so it never drifts from the workflow's own
// guards. Distinct events that report through the same command (verified's
// pass/fail) collapse into one entry.
func commandsFor(t task.Task) []string {
	seen := make(map[string]bool)
	commands := make([]string, 0, len(commandNames))
	for _, kind := range t.ValidEvents() {
		name, ok := commandNames[kind]
		if !ok {
			continue
		}
		command := fmt.Sprintf(name, t.ID)
		if seen[command] {
			continue
		}
		seen[command] = true
		commands = append(commands, command)
	}
	sort.Strings(commands)
	return commands
}
