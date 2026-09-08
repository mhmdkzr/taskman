package task

import "time"

// HasCommitSince reports whether the task's recorded commit was made at or
// after since - i.e. whether it's the commit for the current pass, not a
// stale one from an earlier pass still waiting to be superseded. Exported
// for the "next" slice's own guidance logic (design.md §6/§7), which lives
// outside this package.
func HasCommitSince(t Task, since *time.Time) bool {
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
	return t.Status.Review.State == StagePending && !HasCommitSince(t, t.Status.Verification.CompletedAt)
}
