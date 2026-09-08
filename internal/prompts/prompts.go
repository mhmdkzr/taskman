// Package prompts holds taskman's prompt and message templates as embedded
// Markdown files - never read from disk at runtime. Each template has a
// typed params struct and a Render method, so callers can't pass the wrong
// shape of data to the wrong template. See notes/design/design.md §4.
package prompts

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed *.md
var files embed.FS

func parse(name string) *template.Template {
	return template.Must(template.New(name).ParseFS(files, name))
}

func render(t *template.Template, data any) string {
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		// Every template here is fixed at compile time and every params
		// struct is built by this package's own callers - a failure here
		// is a programming error, not a runtime condition to recover from.
		panic(fmt.Sprintf("prompts: render %s: %v", t.Name(), err))
	}
	return strings.TrimRight(b.String(), "\n")
}

var (
	specifyTmpl         = parse("specify.md")
	implementTmpl       = parse("implement.md")
	fixTmpl             = parse("fix.md")
	reviewTmpl          = parse("review.md")
	commitTmpl          = parse("commit.md")
	dispatchWrapperTmpl = parse("dispatch_wrapper.md")
	runVerifyTmpl       = parse("run_verify.md")
	runMergeTmpl        = parse("run_merge.md")
	waitHumanReviewTmpl = parse("wait_human_review.md")
	waitBlockedTmpl     = parse("wait_blocked.md")
	doneMergedTmpl      = parse("done_merged.md")
	doneAbandonedTmpl   = parse("done_abandoned.md")
	createSummaryTmpl   = parse("create_summary.md")
	taskSummaryTmpl     = parse("task_summary.md")
)

// Specify drafts a specification and done_when for a task - dispatched at
// the specification stage.
type Specify struct {
	Definition string
	References []string
}

func (p Specify) Render() string { return render(specifyTmpl, p) }

// Implement writes the task's diff in its worktree - dispatched at the
// implementation stage.
type Implement struct {
	Definition    string
	Specification string
	DoneWhen      string
	References    []string
}

func (p Implement) Render() string { return render(implementTmpl, p) }

// Fix addresses a reported problem - a failed build check, a rejected
// automated review, or a rejected human review. Reason carries whichever
// of those the caller already formatted as plain text.
type Fix struct {
	Specification string
	DoneWhen      string
	Reason        string
}

func (p Fix) Render() string { return render(fixTmpl, p) }

// Review judges a worktree's diff against its task's acceptance criteria -
// the automated review round inside verification.
type Review struct {
	Specification string
	DoneWhen      string
}

func (p Review) Render() string { return render(reviewTmpl, p) }

// Commit drafts and makes a commit for the task's current diff.
type Commit struct {
	Title         string
	Specification string
}

func (p Commit) Render() string { return render(commitTmpl, p) }

// DispatchWrapper wraps Body (one of the above prompts, already rendered)
// with the worktree/branch/report-back context common to every dispatch -
// design.md §4.
type DispatchWrapper struct {
	Body       string
	Worktree   string
	Branch     string
	ReportWith string
}

func (p DispatchWrapper) Render() string { return render(dispatchWrapperTmpl, p) }

// RunVerify tells the caller to run build checks itself and report the
// results - no prompt is dispatched, since no judgment is needed.
type RunVerify struct {
	TaskID   string
	Worktree string
	Branch   string
}

func (p RunVerify) Render() string { return render(runVerifyTmpl, p) }

// RunMerge tells the caller to run the merge itself and report it.
type RunMerge struct {
	TaskID   string
	Worktree string
	Branch   string
}

func (p RunMerge) Render() string { return render(runMergeTmpl, p) }

// WaitHumanReview briefs the caller that a task is awaiting human review.
type WaitHumanReview struct {
	TaskID       string
	Title        string
	Attempt      int
	CommitHash   string
	Branch       string
	Worktree     string
	WaitingSince string
}

func (p WaitHumanReview) Render() string { return render(waitHumanReviewTmpl, p) }

// WaitBlocked briefs the caller that a task is blocked and needs a human.
type WaitBlocked struct {
	TaskID       string
	Title        string
	Stage        string
	WaitingSince string
	Reason       string
	Worktree     string
	Branch       string
}

func (p WaitBlocked) Render() string { return render(waitBlockedTmpl, p) }

// DoneMerged briefs the caller that a task completed via merge.
type DoneMerged struct {
	TaskID     string
	Title      string
	CommitHash string
}

func (p DoneMerged) Render() string { return render(doneMergedTmpl, p) }

// DoneAbandoned briefs the caller that a task was abandoned.
type DoneAbandoned struct {
	TaskID string
	Title  string
	Reason string
}

func (p DoneAbandoned) Render() string { return render(doneAbandonedTmpl, p) }

// CreateSummary is task create's default (non-JSON) CLI output.
type CreateSummary struct {
	TaskID   string
	Title    string
	Worktree string
	Branch   string
}

func (p CreateSummary) Render() string { return render(createSummaryTmpl, p) }

// TaskSummary is the default (non-JSON) CLI output for any command that
// returns a Task rather than Next's Guidance.
type TaskSummary struct {
	TaskID string
	Title  string
	State  string
	Stage  string
}

func (p TaskSummary) Render() string { return render(taskSummaryTmpl, p) }
