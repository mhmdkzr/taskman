package next

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed *.md
var promptFiles embed.FS

func parse(name string) *template.Template {
	return template.Must(template.New(name).ParseFS(promptFiles, name))
}

func render(t *template.Template, data any) string {
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		panic(fmt.Sprintf("next: render %s: %v", t.Name(), err))
	}
	return strings.TrimRight(b.String(), "\n")
}

var (
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
)

// fixPrompt addresses a reported problem - a failed build check, a rejected
// automated review, or a rejected human review. Reason carries whichever of
// those the caller already formatted as plain text.
type fixPrompt struct {
	Specification string
	DoneWhen      string
	Reason        string
}

func (p fixPrompt) Render() string { return render(fixTmpl, p) }

// reviewPrompt judges a worktree's diff against its task's acceptance
// criteria - the automated review round inside verification.
type reviewPrompt struct {
	Specification string
	DoneWhen      string
}

func (p reviewPrompt) Render() string { return render(reviewTmpl, p) }

// commitPrompt drafts and makes a commit for the task's current diff.
type commitPrompt struct {
	TaskID        string
	Title         string
	Specification string
}

func (p commitPrompt) Render() string { return render(commitTmpl, p) }

// dispatchWrapper wraps body (one of the prompts above, already rendered)
// with the worktree/branch/report-back context common to every dispatch.
type dispatchWrapper struct {
	Body       string
	Worktree   string
	Branch     string
	ReportWith string
}

func (p dispatchWrapper) Render() string { return render(dispatchWrapperTmpl, p) }

// runVerify tells the caller to run build checks itself and report the
// results - no prompt is dispatched, since no judgment is needed.
type runVerify struct {
	TaskID   string
	Worktree string
	Branch   string
}

func (p runVerify) Render() string { return render(runVerifyTmpl, p) }

// runMerge tells the caller to run the merge itself and report it - or, for
// a task created with --trunk, that there's nothing to merge at all.
type runMerge struct {
	TaskID   string
	Worktree string
	Branch   string
	Trunk    bool
}

func (p runMerge) Render() string { return render(runMergeTmpl, p) }

// waitHumanReview briefs the caller that a task is awaiting human review.
type waitHumanReview struct {
	TaskID       string
	Title        string
	Attempt      int
	CommitHash   string
	Branch       string
	Worktree     string
	WaitingSince string
}

func (p waitHumanReview) Render() string { return render(waitHumanReviewTmpl, p) }

// waitBlocked briefs the caller that a task is blocked and needs a human.
type waitBlocked struct {
	TaskID       string
	Title        string
	Stage        string
	WaitingSince string
	Reason       string
	Worktree     string
	Branch       string
}

func (p waitBlocked) Render() string { return render(waitBlockedTmpl, p) }

// doneMerged briefs the caller that a task completed via merge - or, for a
// task created with --trunk, that it completed in place with no merge.
type doneMerged struct {
	TaskID     string
	Title      string
	Branch     string
	CommitHash string
	Trunk      bool
}

func (p doneMerged) Render() string { return render(doneMergedTmpl, p) }

// doneAbandoned briefs the caller that a task was abandoned.
type doneAbandoned struct {
	TaskID string
	Title  string
	Reason string
}

func (p doneAbandoned) Render() string { return render(doneAbandonedTmpl, p) }
