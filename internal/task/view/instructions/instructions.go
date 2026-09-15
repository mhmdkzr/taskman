// Package instructions projects a task's current workflow position into
// actionable instructions: a per-state rendered message plus the command(s)
// that would currently report an outcome. It is presentation - derived from
// an already-loaded task, never persisted - consumed by view's JSON and CLI
// renderers so every command's output self-describes what to do next.
package instructions

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	tmpl "text/template"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
)

//go:embed instructions.md
var instructionsTemplate string

var tpl = tmpl.Must(tmpl.New("instructions.md").Parse(instructionsTemplate))

// Instructions is a task's current instruction, rendered as a message for
// whichever caller (human or agent) is meant to act on it, plus the
// command(s) that would currently report an outcome.
type Instructions struct {
	TaskID   uuid.UUID              `json:"task_id"`
	State    task.TaskState         `json:"state"`
	Action   task.InstructionAction `json:"action"`
	Message  string                 `json:"message"`
	Commands []string               `json:"commands,omitempty"`
}

// Project renders t's current state's instructions.
func Project(t task.Task) (Instructions, error) {
	instruction := t.Instruction()
	instructions := Instructions{
		TaskID:   t.ID,
		State:    instruction.State,
		Action:   instruction.Action,
		Commands: CommandsFor(t),
	}
	var buf bytes.Buffer
	data := view{Task: t, Reason: reasonFor(t, instruction.State)}
	if err := tpl.ExecuteTemplate(&buf, string(instruction.State), data); err != nil {
		return Instructions{}, fmt.Errorf("render instructions: %w", err)
	}
	instructions.Message = strings.TrimSpace(buf.String())
	return instructions, nil
}

// view is the data instructions.md's per-state templates render against.
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

// AutoFixNote describes the auto-fix policy governing the current fix state -
// whether fixing is automatic, how many rounds it may spend, and whether it
// should run in a subagent - or "" when the current state is not a fix state
// or its gate has no auto-fix configured.
func (v view) AutoFixNote() string {
	if v.Task.Implementation == nil {
		return ""
	}
	var (
		config task.AutoFix
		gate   string
	)
	switch v.Task.State() {
	case task.StateFixVerificationFailure:
		config, gate = v.Task.Implementation.Verification.AutoFix, "verification"
	case task.StateFixAutomatedReviewFindings:
		config, gate = v.Task.Implementation.Review.Agent.AutoFix, "the automated review"
	case task.StateFixHumanReviewFindings:
		config, gate = v.Task.Implementation.Review.Human.AutoFix, "the human review"
	default:
		return ""
	}
	if !config.Enabled {
		return ""
	}
	note := "Auto-fix is enabled for " + gate + "."
	if config.MaxRounds > 0 {
		note += fmt.Sprintf(" It is capped at %d round(s); exceeding the cap blocks the task.", config.MaxRounds)
	}
	if config.UseSubagent {
		note += " Run the fix in a subagent."
	}
	return note
}

// reasonFor explains why a state needs feedback - the specification
// rejection a resubmission must address, or the failure/rejection a fix
// state's dispatched agent needs to address. Every other state's
// instructions is self-explanatory from the task alone.
func reasonFor(t task.Task, state task.TaskState) string {
	switch state {
	case task.StateSpecify:
		return specificationRejectionReason(t)
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

// specificationRejectionReason explains what a specification's most recent
// rejections asked for - the agent gate's findings and the human gate's
// comment, latest per gate. A task without a submitted specification, or
// one without a rejection, returns "" and the template's feedback block is
// omitted.
func specificationRejectionReason(t task.Task) string {
	var sections []string
	if t.Specification != nil {
		if results := t.Specification.Review.Agent.Results; len(results) > 0 {
			if last := results[len(results)-1]; !last.Approved {
				var b strings.Builder
				b.WriteString("Automated specification review rejected this plan:\n")
				for _, finding := range last.Findings {
					fmt.Fprintf(&b, "- %s: %s\n", finding.Location, finding.Detail)
				}
				sections = append(sections, strings.TrimRight(b.String(), "\n"))
			}
		}
		if results := t.Specification.Review.Human.Results; len(results) > 0 {
			if last := results[len(results)-1]; !last.Approved && last.Comment != "" {
				sections = append(sections, "A human rejected this specification: "+last.Comment)
			}
		}
	}
	return strings.Join(sections, "\n")
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
