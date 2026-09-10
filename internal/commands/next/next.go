// Package next renders the pure workflow's current instruction for CLI and MCP callers.
package next

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mhmdkzr/taskman/internal/commands/implement"
	"github.com/mhmdkzr/taskman/internal/commands/specify"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

type Action string

const (
	ActionDispatch Action = "dispatch"
	ActionRun      Action = "run"
	ActionWait     Action = "wait"
	ActionDone     Action = "done"
)

type Guidance struct {
	TaskID     string `json:"task_id"`
	Action     Action `json:"action"`
	Message    string `json:"message"`
	ReportWith string `json:"report_with,omitempty"`
}

const timeLayout = "2006-01-02 15:04 MST"

type Request struct {
	ID string `json:"id" jsonschema:"the task id to inspect"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

func Next(tasksDir string, req Request) (Guidance, error) {
	if err := req.validate(); err != nil {
		return Guidance{}, fmt.Errorf("next: %w", err)
	}
	t, err := store.Read(tasksDir, req.ID)
	if err != nil {
		return Guidance{}, fmt.Errorf("read task: %w", err)
	}
	instruction, err := task.Next(t)
	if err != nil {
		return Guidance{}, fmt.Errorf("next: %w", err)
	}

	var guidance Guidance
	switch instruction.Kind {
	case task.InstructionSpecify:
		guidance = guideSpecify(t)
	case task.InstructionImplement:
		guidance = guideImplement(t)
	case task.InstructionVerify:
		guidance = guideRunVerify(t)
	case task.InstructionFixVerificationFailure:
		guidance = guideFix(t, verificationFailureReason(t))
	case task.InstructionFixAutomatedReviewFindings:
		guidance = guideFix(t, automatedReviewFindingsReason(t))
	case task.InstructionAutomatedReview:
		guidance = guideAutomatedReview(t)
	case task.InstructionCommit:
		guidance = guideDraftCommit(t)
	case task.InstructionHumanReview:
		guidance = guideWaitHumanReview(t)
	case task.InstructionFixHumanReviewFindings:
		guidance = guideFix(t, humanReviewFindingsReason(t))
	case task.InstructionMerge:
		guidance = guideMerge(t)
	case task.InstructionBlocked:
		guidance = guideBlocked(t)
	case task.InstructionCompleted:
		guidance = guideCompleted(t)
	case task.InstructionAbandoned:
		guidance = guideAbandoned(t)
	default:
		return Guidance{}, fmt.Errorf("task %s: unsupported instruction %q", t.ID, instruction.Kind)
	}
	guidance.Action = Action(instruction.Action)
	return guidance, nil
}

func specifyReport(id string) (string, string) {
	short := "specify " + id
	return short, short + " --result <text> --done-when <text>"
}

func verifyReport(id string) (string, string) {
	short := "verify " + id
	return short, short + " --check <name>=<ok|error> ... [--output <text>]"
}

func reviewRecordReport(id string) (string, string) {
	short := "review record " + id
	return short, short + " --approved <bool> [--finding <file>=<text> ...]"
}

func commitReport(id string) (string, string) {
	short := "commit " + id
	return short, short + " [--commit <hash>]"
}

func guideSpecify(t task.Task) Guidance {
	body := specify.Prompt{Definition: t.Definition, References: t.References}.Render()
	short, full := specifyReport(t.ID)
	return dispatch(t, body, short, full)
}

func guideImplement(t task.Task) Guidance {
	body := implement.Prompt{
		Definition:    t.Definition,
		Specification: t.Specification,
		DoneWhen:      t.DoneWhen,
		References:    t.References,
	}.Render()
	short := "implement " + t.ID
	return dispatch(t, body, short, short)
}

func guideRunVerify(t task.Task) Guidance {
	message := runVerify{TaskID: t.ID, Worktree: t.Git.Worktree, Branch: t.Git.Branch}.Render()
	short, _ := verifyReport(t.ID)
	return Guidance{TaskID: t.ID, Action: ActionRun, Message: message, ReportWith: short}
}

func guideFix(t task.Task, reason string) Guidance {
	body := fixPrompt{Specification: t.Specification, DoneWhen: t.DoneWhen, Reason: reason}.Render()
	short, full := verifyReport(t.ID)
	return dispatch(t, body, short, full)
}

func guideAutomatedReview(t task.Task) Guidance {
	body := reviewPrompt{Specification: t.Specification, DoneWhen: t.DoneWhen}.Render()
	short, full := reviewRecordReport(t.ID)
	return dispatch(t, body, short, full)
}

func guideDraftCommit(t task.Task) Guidance {
	body := commitPrompt{TaskID: t.ID, Title: t.Title, Specification: t.Specification}.Render()
	short, full := commitReport(t.ID)
	return dispatch(t, body, short, full)
}

func guideWaitHumanReview(t task.Task) Guidance {
	hash := ""
	if t.Git.Commit != nil {
		hash = t.Git.Commit.Hash
	}
	waitingSince := time.Time{}
	if len(t.Reviews) > 0 {
		waitingSince = t.Reviews[len(t.Reviews)-1].CreatedAt
	}
	if waitingSince.IsZero() && t.Git.Commit != nil {
		waitingSince = t.Git.Commit.At
	}
	message := waitHumanReview{
		TaskID: t.ID, Title: t.Title, Attempt: max(len(t.Reviews), 1), CommitHash: hash,
		Branch: t.Git.Branch, Worktree: t.Git.Worktree, WaitingSince: formatTime(waitingSince),
	}.Render()
	return Guidance{TaskID: t.ID, Action: ActionWait, Message: message}
}

func guideMerge(t task.Task) Guidance {
	message := runMerge{TaskID: t.ID, Worktree: t.Git.Worktree, Branch: t.Git.Branch, Trunk: t.Git.Trunk}.Render()
	return Guidance{TaskID: t.ID, Action: ActionRun, Message: message, ReportWith: "merge " + t.ID}
}

func guideBlocked(t task.Task) Guidance {
	stage, reason, at := "", "", ""
	if t.Blocked != nil {
		stage, reason, at = t.Blocked.Stage, t.Blocked.Reason, formatTime(t.Blocked.At)
	}
	message := waitBlocked{
		TaskID:       t.ID,
		Title:        t.Title,
		Stage:        stage,
		WaitingSince: at,
		Reason:       reason,
		Worktree:     t.Git.Worktree,
		Branch:       t.Git.Branch,
	}.Render()
	return Guidance{TaskID: t.ID, Action: ActionWait, Message: message}
}

func guideCompleted(t task.Task) Guidance {
	hash := ""
	if t.Git.Commit != nil {
		hash = t.Git.Commit.Hash
	}
	message := doneMerged{
		TaskID:     t.ID,
		Title:      t.Title,
		Branch:     t.Git.Branch,
		CommitHash: hash,
		Trunk:      t.Git.Trunk,
	}.Render()
	return Guidance{TaskID: t.ID, Action: ActionDone, Message: message}
}

func guideAbandoned(t task.Task) Guidance {
	message := doneAbandoned{TaskID: t.ID, Title: t.Title, Reason: t.FailureReason}.Render()
	return Guidance{TaskID: t.ID, Action: ActionDone, Message: message}
}

func automatedReviewFindingsReason(t task.Task) string {
	lastReview := lastReview(t)
	if lastReview != nil {
		return "Automated review rejected this attempt:\n" + task.SummarizeFindings(lastReview.Findings)
	}
	return "The automated reviewer reported findings that need to be fixed."
}

func humanReviewFindingsReason(t task.Task) string {
	lastVerification := lastVerification(t)
	lastHuman := lastHumanReview(t)
	if lastVerification != nil && !lastVerification.Passed() &&
		(lastHuman == nil || lastVerification.CreatedAt.After(lastHuman.At)) {
		return verificationFailureReason(t)
	}
	if lastHuman != nil {
		return "A human rejected this task's review: " + lastHuman.Comment
	}
	return "The human-review changes need another verification pass."
}

func verificationFailureReason(t task.Task) string {
	verification := lastVerification(t)
	if verification == nil {
		return "Verification failed."
	}
	failed := make([]string, 0, len(verification.Checks))
	for name, result := range verification.Checks {
		if result != task.CheckOK {
			failed = append(failed, name)
		}
	}
	sort.Strings(failed)
	reason := "Verification failed"
	if len(failed) > 0 {
		reason += " (" + strings.Join(failed, ", ") + ")"
	}
	if verification.Output != "" {
		reason += ":\n" + verification.Output
		return reason
	}
	return reason + "."
}

func dispatch(t task.Task, body, reportShort, reportFull string) Guidance {
	message := dispatchWrapper{
		Body:       body,
		Worktree:   t.Git.Worktree,
		Branch:     t.Git.Branch,
		ReportWith: reportFull,
	}.Render()
	return Guidance{TaskID: t.ID, Action: ActionDispatch, Message: message, ReportWith: reportShort}
}

func lastVerification(t task.Task) *task.Verification {
	if len(t.Verifications) == 0 {
		return nil
	}
	return &t.Verifications[len(t.Verifications)-1]
}

func lastReview(t task.Task) *task.Review {
	if len(t.Reviews) == 0 {
		return nil
	}
	return &t.Reviews[len(t.Reviews)-1]
}

func lastHumanReview(t task.Task) *task.HumanReview {
	if len(t.HumanReviews) == 0 {
		return nil
	}
	return &t.HumanReviews[len(t.HumanReviews)-1]
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(timeLayout)
}
