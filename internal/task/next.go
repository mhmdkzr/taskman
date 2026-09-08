package task

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/loop/internal/prompts"
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
func Next(repo *Repo, id string) (Guidance, error) {
	t, err := repo.Get(id)
	if err != nil {
		return Guidance{}, err
	}

	switch t.State { //nolint:exhaustive // created/started fall through to the per-stage switch below
	case StateCompleted:
		hash := ""
		if t.Git.Commit != nil {
			hash = t.Git.Commit.Hash
		}
		message := prompts.DoneMerged{TaskID: t.ID, Title: t.Title, CommitHash: hash}.Render()
		return Guidance{TaskID: t.ID, Action: ActionDone, Message: message}, nil
	case StateFailed:
		message := prompts.DoneAbandoned{TaskID: t.ID, Title: t.Title, Reason: t.FailureReason}.Render()
		return Guidance{TaskID: t.ID, Action: ActionDone, Message: message}, nil
	case StateBlocked:
		return guideBlocked(t), nil
	}

	switch {
	case t.Status.Specification.State != StageDone:
		return guideSpecify(t), nil
	case t.Status.Implementation.State != StageDone:
		return guideImplement(t), nil
	case t.Status.Verification.State != StageDone:
		return guideVerification(t), nil
	case t.Status.Review.State == StagePending:
		if !hasCommitSince(t, t.Status.Verification.CompletedAt) {
			return guideDraftCommit(t), nil
		}
		return guideWaitHumanReview(t), nil
	case t.Status.Review.State == StageInProgress:
		return guideReviewRejectRecovery(t)
	case t.Status.Merge.State != StageDone:
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

func guideSpecify(t Task) Guidance {
	body := prompts.Specify{Definition: t.Definition, References: t.References}.Render()
	short, full := specifyReport(t.ID)
	return dispatch(t, body, short, full)
}

func guideImplement(t Task) Guidance {
	body := prompts.Implement{
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
func guideVerification(t Task) Guidance {
	lastVerify := lastVerification(t)
	lastReview := lastReview(t)
	verifyShort, verifyFull := verifyReport(t.ID)

	switch {
	case lastVerify == nil:
		// Implementation just finished; nothing has failed yet, so the
		// first build check is mechanical, not a judgment call.
		return guideRunVerify(t)
	case !lastVerify.Passed():
		body := prompts.Fix{
			Specification: t.Specification,
			DoneWhen:      t.DoneWhen,
			Reason:        "Build check failed:\n" + lastVerify.Output,
		}.Render()
		return dispatch(t, body, verifyShort, verifyFull)
	case lastReview == nil || lastReview.CreatedAt.Before(lastVerify.CreatedAt):
		// This verify pass hasn't been reviewed yet.
		body := prompts.Review{Specification: t.Specification, DoneWhen: t.DoneWhen}.Render()
		short, full := reviewRecordReport(t.ID)
		return dispatch(t, body, short, full)
	default:
		// lastReview happened after lastVerify and rejected it (if it had
		// approved, verification.state would already be done and this
		// function wouldn't have been called) - fix and re-check.
		body := prompts.Fix{
			Specification: t.Specification,
			DoneWhen:      t.DoneWhen,
			Reason:        "Automated review rejected this attempt:\n" + summarizeFindings(lastReview.Findings),
		}.Render()
		return dispatch(t, body, verifyShort, verifyFull)
	}
}

// guideReviewRejectRecovery handles the human-rejection recovery cycle -
// design.md §6's "Review-reject recovery". It never re-dispatches an
// automated review; the human is the reviewer for the rest of this cycle.
func guideReviewRejectRecovery(t Task) (Guidance, error) {
	if len(t.HumanReviews) == 0 {
		return Guidance{}, fmt.Errorf("task %s: review in_progress with no recorded rejection", t.ID)
	}
	lastHuman := t.HumanReviews[len(t.HumanReviews)-1]
	lastVerify := lastVerification(t)
	verifyShort, verifyFull := verifyReport(t.ID)

	switch {
	case lastVerify == nil || !lastVerify.CreatedAt.After(lastHuman.At):
		body := prompts.Fix{
			Specification: t.Specification,
			DoneWhen:      t.DoneWhen,
			Reason:        "A human rejected this task's review: " + lastHuman.Comment,
		}.Render()
		return dispatch(t, body, verifyShort, verifyFull), nil
	case !lastVerify.Passed():
		body := prompts.Fix{
			Specification: t.Specification,
			DoneWhen:      t.DoneWhen,
			Reason:        "Build check failed:\n" + lastVerify.Output,
		}.Render()
		return dispatch(t, body, verifyShort, verifyFull), nil
	case !hasCommitSince(t, &lastHuman.At):
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

func guideRunVerify(t Task) Guidance {
	message := prompts.RunVerify{TaskID: t.ID, Worktree: t.Git.Worktree, Branch: t.Git.Branch}.Render()
	short, _ := verifyReport(t.ID)
	return Guidance{TaskID: t.ID, Action: ActionRun, Message: message, ReportWith: short}
}

func guideMerge(t Task) Guidance {
	message := prompts.RunMerge{TaskID: t.ID, Worktree: t.Git.Worktree, Branch: t.Git.Branch}.Render()
	return Guidance{TaskID: t.ID, Action: ActionRun, Message: message, ReportWith: "task merge " + t.ID}
}

func guideDraftCommit(t Task) Guidance {
	body := prompts.Commit{Title: t.Title, Specification: t.Specification}.Render()
	short, full := commitReport(t.ID)
	return dispatch(t, body, short, full)
}

func guideWaitHumanReview(t Task) Guidance {
	hash := ""
	if t.Git.Commit != nil {
		hash = t.Git.Commit.Hash
	}
	message := prompts.WaitHumanReview{
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

func guideBlocked(t Task) Guidance {
	stage, reason, at := "", "", ""
	if t.Blocked != nil {
		stage, reason = t.Blocked.Stage, t.Blocked.Reason
		at = t.Blocked.At.Format(timeLayout)
	}
	message := prompts.WaitBlocked{
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
func dispatch(t Task, body, reportShort, reportFull string) Guidance {
	message := prompts.DispatchWrapper{
		Body:       body,
		Worktree:   t.Git.Worktree,
		Branch:     t.Git.Branch,
		ReportWith: reportFull,
	}.Render()
	return Guidance{TaskID: t.ID, Action: ActionDispatch, Message: message, ReportWith: reportShort}
}

func lastVerification(t Task) *Verification {
	if len(t.Verifications) == 0 {
		return nil
	}
	return &t.Verifications[len(t.Verifications)-1]
}

func lastReview(t Task) *Review {
	if len(t.Reviews) == 0 {
		return nil
	}
	return &t.Reviews[len(t.Reviews)-1]
}

// hasCommitSince reports whether the task's recorded commit was made at or
// after since - i.e. whether it's the commit for the current pass, not a
// stale one from an earlier pass still waiting to be superseded.
func hasCommitSince(t Task, since *time.Time) bool {
	if t.Git.Commit == nil || since == nil {
		return false
	}
	return !t.Git.Commit.At.Before(*since)
}

// NeedsFreshCommit reports whether t's review stage is pending but still
// waiting on a commit for the pass that just got it there - i.e. whether
// Next would dispatch drafting a commit rather than say "awaiting human
// review". Exported for callers rendering their own summary of a task
// (e.g. the CLI's default, non-JSON output) that want to describe this
// distinction without reimplementing Next's own logic.
func NeedsFreshCommit(t Task) bool {
	return t.Status.Review.State == StagePending && !hasCommitSince(t, t.Status.Verification.CompletedAt)
}

func formatTime(candidates ...*time.Time) string {
	for _, c := range candidates {
		if c != nil {
			return c.Format(timeLayout)
		}
	}
	return ""
}
