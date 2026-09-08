// Package next owns the "next" command: task next inspects a task's
// current state and tells the caller what to do about it - design.md §6/§7.
// It is the one command that legitimately depends on every other stage,
// since guiding a task means knowing what every stage's own commands would
// accept next.
package next

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/cli/task/implement"
	"github.com/mhmdkzr/taskman/internal/cli/task/specify"
	"github.com/mhmdkzr/taskman/internal/task"
)

// Action is Next's coarse signal for a caller with no way to read prose -
// specifically loop, which has no LLM. design.md §7.
type Action string

const (
	ActionDispatch Action = "dispatch"
	ActionRun      Action = "run"
	ActionWait     Action = "wait"
	ActionDone     Action = "done"
)

// Guidance is task next's response - design.md §7's "task next: taskman
// talks, the caller listens".
type Guidance struct {
	TaskID     string `json:"task_id"`
	Action     Action `json:"action"`
	Message    string `json:"message"`
	ReportWith string `json:"report_with,omitempty"`
}

const timeLayout = "2006-01-02 15:04 MST"

// Next inspects task id's current state and returns what the caller should
// do about it - design.md §6's "task next: what tells the caller what to
// do".
func Next(tasksDir, id string) (Guidance, error) {
	t, err := task.ReadTask(tasksDir, id)
	if err != nil {
		return Guidance{}, fmt.Errorf("read task: %w", err)
	}

	switch t.State { //nolint:exhaustive // created/started fall through to the per-stage switch below
	case task.StateCompleted:
		hash := ""
		if t.Git.Commit != nil {
			hash = t.Git.Commit.Hash
		}
		message := doneMerged{
			TaskID: t.ID, Title: t.Title, Branch: t.Git.Branch, CommitHash: hash, Trunk: t.Git.Trunk,
		}.Render()
		return Guidance{TaskID: t.ID, Action: ActionDone, Message: message}, nil
	case task.StateFailed:
		message := doneAbandoned{TaskID: t.ID, Title: t.Title, Reason: t.FailureReason}.Render()
		return Guidance{TaskID: t.ID, Action: ActionDone, Message: message}, nil
	case task.StateBlocked:
		return guideBlocked(t), nil
	}

	switch {
	case t.Status.Specification.State != task.StageDone:
		return guideSpecify(t), nil
	case t.Status.Implementation.State != task.StageDone:
		return guideImplement(t), nil
	case t.Status.Verification.State != task.StageDone:
		return guideVerification(t), nil
	case t.Status.Review.State == task.StagePending:
		if !task.HasCommitSince(t, t.Status.Verification.CompletedAt) {
			return guideDraftCommit(t), nil
		}
		return guideWaitHumanReview(t), nil
	case t.Status.Review.State == task.StageInProgress:
		return guideReviewRejectRecovery(t)
	case t.Status.Merge.State != task.StageDone:
		return guideMerge(t), nil
	}
	// review.state == done, merge.state == done, but task.State never
	// reached completed - shouldn't happen via the command layer, but fail
	// safe rather than pretend there's nothing to do.
	return Guidance{}, fmt.Errorf("task %s: no guidance for its current state", id)
}

// Each of these pairs a short command (for Guidance.ReportWith, and for
// loop, which can't read the fuller instructional form out of prose) with
// the fuller form shown inside the dispatched message itself.
func specifyReport(id string) (string, string) {
	short := "task specify " + id
	return short, short + " --result <text> --done-when <text>"
}

func verifyReport(id string) (string, string) {
	short := "task verify " + id
	return short, short + " --check <name>=<ok|error> ... [--output <text>]"
}

func reviewRecordReport(id string) (string, string) {
	short := "task review record " + id
	return short, short + " --approved <bool> [--finding <file>=<text> ...]"
}

func commitReport(id string) (string, string) {
	short := "task commit " + id
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
	short := "task implement " + t.ID
	return dispatch(t, body, short, short)
}

// guideVerification handles every step of the build-check-fix-review loop
// inside the verification stage - design.md §6's "The verification stage,
// in full". It's only ever called while verification.state != done, so it
// never itself dispatches the commit - that's the outer switch's job, the
// moment verification.state (and, simultaneously, review.state: pending)
// first appear.
func guideVerification(t task.Task) Guidance {
	lastVerify := lastVerification(t)
	lastReview := lastReview(t)
	verifyShort, verifyFull := verifyReport(t.ID)

	switch {
	case lastVerify == nil:
		// Implementation just finished; nothing has failed yet, so the
		// first build check is mechanical, not a judgment call.
		return guideRunVerify(t)
	case !lastVerify.Passed():
		body := fixPrompt{
			Specification: t.Specification,
			DoneWhen:      t.DoneWhen,
			Reason:        "Build check failed:\n" + lastVerify.Output,
		}.Render()
		return dispatch(t, body, verifyShort, verifyFull)
	case lastReview == nil || lastReview.CreatedAt.Before(lastVerify.CreatedAt):
		// This verify pass hasn't been reviewed yet.
		body := reviewPrompt{Specification: t.Specification, DoneWhen: t.DoneWhen}.Render()
		short, full := reviewRecordReport(t.ID)
		return dispatch(t, body, short, full)
	default:
		// lastReview happened after lastVerify and rejected it (if it had
		// approved, verification.state would already be done and this
		// function wouldn't have been called) - fix and re-check.
		body := fixPrompt{
			Specification: t.Specification,
			DoneWhen:      t.DoneWhen,
			Reason:        "Automated review rejected this attempt:\n" + task.SummarizeFindings(lastReview.Findings),
		}.Render()
		return dispatch(t, body, verifyShort, verifyFull)
	}
}

// guideReviewRejectRecovery handles the human-rejection recovery cycle -
// design.md §6's "Review-reject recovery". It never re-dispatches an
// automated review; the human is the reviewer for the rest of this cycle.
func guideReviewRejectRecovery(t task.Task) (Guidance, error) {
	if len(t.HumanReviews) == 0 {
		return Guidance{}, fmt.Errorf("task %s: review in_progress with no recorded rejection", t.ID)
	}
	lastHuman := t.HumanReviews[len(t.HumanReviews)-1]
	lastVerify := lastVerification(t)
	verifyShort, verifyFull := verifyReport(t.ID)

	switch {
	case lastVerify == nil || !lastVerify.CreatedAt.After(lastHuman.At):
		body := fixPrompt{
			Specification: t.Specification,
			DoneWhen:      t.DoneWhen,
			Reason:        "A human rejected this task's review: " + lastHuman.Comment,
		}.Render()
		return dispatch(t, body, verifyShort, verifyFull), nil
	case !lastVerify.Passed():
		body := fixPrompt{
			Specification: t.Specification,
			DoneWhen:      t.DoneWhen,
			Reason:        "Build check failed:\n" + lastVerify.Output,
		}.Render()
		return dispatch(t, body, verifyShort, verifyFull), nil
	case !task.HasCommitSince(t, &lastHuman.At):
		return guideDraftCommit(t), nil
	default:
		// A fresh commit already exists since the rejection - task commit
		// would have flipped review.state back to pending, so this branch
		// shouldn't be reachable through the command layer.
		return Guidance{}, fmt.Errorf(
			"task %s: review-reject recovery already cleared but review.state is still in_progress",
			t.ID,
		)
	}
}

func guideRunVerify(t task.Task) Guidance {
	message := runVerify{TaskID: t.ID, Worktree: t.Git.Worktree, Branch: t.Git.Branch}.Render()
	short, _ := verifyReport(t.ID)
	return Guidance{TaskID: t.ID, Action: ActionRun, Message: message, ReportWith: short}
}

func guideMerge(t task.Task) Guidance {
	message := runMerge{
		TaskID: t.ID, Worktree: t.Git.Worktree, Branch: t.Git.Branch, Trunk: t.Git.Trunk,
	}.Render()
	return Guidance{TaskID: t.ID, Action: ActionRun, Message: message, ReportWith: "task merge " + t.ID}
}

func guideDraftCommit(t task.Task) Guidance {
	body := commitPrompt{Title: t.Title, Specification: t.Specification}.Render()
	short, full := commitReport(t.ID)
	return dispatch(t, body, short, full)
}

func guideWaitHumanReview(t task.Task) Guidance {
	hash := ""
	if t.Git.Commit != nil {
		hash = t.Git.Commit.Hash
	}
	message := waitHumanReview{
		TaskID:       t.ID,
		Title:        t.Title,
		Attempt:      t.Status.Verification.Attempts,
		CommitHash:   hash,
		Branch:       t.Git.Branch,
		Worktree:     t.Git.Worktree,
		WaitingSince: formatTime(t.Status.Review.CompletedAt, t.Status.Verification.CompletedAt),
	}.Render()
	return Guidance{TaskID: t.ID, Action: ActionWait, Message: message}
}

func guideBlocked(t task.Task) Guidance {
	stage, reason, at := "", "", ""
	if t.Blocked != nil {
		stage, reason = t.Blocked.Stage, t.Blocked.Reason
		at = t.Blocked.At.Format(timeLayout)
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

// dispatch wraps body with the worktree/branch/report-back context common
// to every dispatch. reportFull (the fuller, flag-annotated form) goes into
// the message text; reportShort goes into Guidance.ReportWith, for loop.
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

func formatTime(candidates ...*time.Time) string {
	for _, c := range candidates {
		if c != nil {
			return c.Format(timeLayout)
		}
	}
	return ""
}
