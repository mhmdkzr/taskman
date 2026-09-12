package task

import "time"

// EventKind identifies a TaskEvent for workflow-table lookups.
type EventKind string

const (
	EventSpecificationSubmitted  EventKind = "specification_submitted"
	EventImplementationCompleted EventKind = "implementation_completed"
	EventVerificationPassed      EventKind = "verification_passed"
	EventVerificationFailed      EventKind = "verification_failed"
	EventCommitRecorded          EventKind = "commit_recorded"
	EventMergeCompleted          EventKind = "merge_completed"
	EventEscalated               EventKind = "escalated"
	EventAbandoned               EventKind = "abandoned"

	// EventSpecificationReviewAgentApproved is the first of the four review
	// gates' own event kinds - stage (specification/implementation) x reviewer
	// (agent/human) - rather than one event shared across stages, so an
	// event's Go type alone always says which gate it reports on. The other
	// seven review-gate kinds below follow the same pattern.
	EventSpecificationReviewAgentApproved  EventKind = "specification_review_agent_approved"
	EventSpecificationReviewAgentRejected  EventKind = "specification_review_agent_rejected"
	EventSpecificationReviewHumanApproved  EventKind = "specification_review_human_approved"
	EventSpecificationReviewHumanRejected  EventKind = "specification_review_human_rejected"
	EventImplementationReviewAgentApproved EventKind = "implementation_review_agent_approved"
	EventImplementationReviewAgentRejected EventKind = "implementation_review_agent_rejected"
	EventImplementationReviewHumanApproved EventKind = "implementation_review_human_approved"
	EventImplementationReviewHumanRejected EventKind = "implementation_review_human_rejected"
)

// TaskEvent is the closed set of facts that may progress a task's state.
// Every event carries its own At: when it occurred, as observed by the
// caller (a clock reading, or a Git commit's own timestamp) - the pure core
// never reads a clock itself.
type TaskEvent interface {
	Kind() EventKind
	OccurredAt() time.Time
}

type SpecificationSubmitted struct {
	Specification Specification `json:"specification"`
	At            time.Time     `json:"at"`
}

type ImplementationCompleted struct {
	Implementation Implementation `json:"implementation"`
	At             time.Time      `json:"at"`
}

type VerificationPassed struct {
	Checks Checks    `json:"checks"`
	Output string    `json:"output,omitempty"`
	At     time.Time `json:"at"`
}

type VerificationFailed struct {
	Checks Checks    `json:"checks"`
	Output string    `json:"output,omitempty"`
	At     time.Time `json:"at"`
}

// CommitRecorded is the event of taskman observing a commit. At is when the
// event was recorded; Commit.At is the commit's own Git timestamp.
type CommitRecorded struct {
	Commit GitCommit `json:"commit"`
	At     time.Time `json:"at"`
}

// MergeCompleted is the event of taskman observing a merge. At is when the
// event was recorded; Merge.At is the merge's own Git timestamp.
type MergeCompleted struct {
	Merge GitMerge  `json:"merge"`
	At    time.Time `json:"at"`
}

type Escalated struct {
	Stage  string    `json:"stage"`
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}

type Abandoned struct {
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}

type SpecificationReviewAgentApproved struct {
	Comment string    `json:"comment,omitempty"`
	At      time.Time `json:"at"`
}

type SpecificationReviewAgentRejected struct {
	Findings []Finding `json:"findings"`
	At       time.Time `json:"at"`
}

type SpecificationReviewHumanApproved struct {
	Comment string    `json:"comment,omitempty"`
	At      time.Time `json:"at"`
}

type SpecificationReviewHumanRejected struct {
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}

type ImplementationReviewAgentApproved struct {
	Comment string    `json:"comment,omitempty"`
	At      time.Time `json:"at"`
}

type ImplementationReviewAgentRejected struct {
	Findings []Finding `json:"findings"`
	At       time.Time `json:"at"`
}

type ImplementationReviewHumanApproved struct {
	Comment string    `json:"comment,omitempty"`
	At      time.Time `json:"at"`
}

type ImplementationReviewHumanRejected struct {
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}

func (e SpecificationSubmitted) Kind() EventKind  { return EventSpecificationSubmitted }
func (e ImplementationCompleted) Kind() EventKind { return EventImplementationCompleted }
func (e VerificationPassed) Kind() EventKind      { return EventVerificationPassed }
func (e VerificationFailed) Kind() EventKind      { return EventVerificationFailed }
func (e CommitRecorded) Kind() EventKind          { return EventCommitRecorded }
func (e MergeCompleted) Kind() EventKind          { return EventMergeCompleted }
func (e Escalated) Kind() EventKind               { return EventEscalated }
func (e Abandoned) Kind() EventKind               { return EventAbandoned }
func (e SpecificationReviewAgentApproved) Kind() EventKind {
	return EventSpecificationReviewAgentApproved
}

func (e SpecificationReviewAgentRejected) Kind() EventKind {
	return EventSpecificationReviewAgentRejected
}

func (e SpecificationReviewHumanApproved) Kind() EventKind {
	return EventSpecificationReviewHumanApproved
}

func (e SpecificationReviewHumanRejected) Kind() EventKind {
	return EventSpecificationReviewHumanRejected
}

func (e ImplementationReviewAgentApproved) Kind() EventKind {
	return EventImplementationReviewAgentApproved
}

func (e ImplementationReviewAgentRejected) Kind() EventKind {
	return EventImplementationReviewAgentRejected
}

func (e ImplementationReviewHumanApproved) Kind() EventKind {
	return EventImplementationReviewHumanApproved
}

func (e ImplementationReviewHumanRejected) Kind() EventKind {
	return EventImplementationReviewHumanRejected
}

func (e SpecificationSubmitted) OccurredAt() time.Time            { return e.At }
func (e ImplementationCompleted) OccurredAt() time.Time           { return e.At }
func (e VerificationPassed) OccurredAt() time.Time                { return e.At }
func (e VerificationFailed) OccurredAt() time.Time                { return e.At }
func (e CommitRecorded) OccurredAt() time.Time                    { return e.At }
func (e MergeCompleted) OccurredAt() time.Time                    { return e.At }
func (e Escalated) OccurredAt() time.Time                         { return e.At }
func (e Abandoned) OccurredAt() time.Time                         { return e.At }
func (e SpecificationReviewAgentApproved) OccurredAt() time.Time  { return e.At }
func (e SpecificationReviewAgentRejected) OccurredAt() time.Time  { return e.At }
func (e SpecificationReviewHumanApproved) OccurredAt() time.Time  { return e.At }
func (e SpecificationReviewHumanRejected) OccurredAt() time.Time  { return e.At }
func (e ImplementationReviewAgentApproved) OccurredAt() time.Time { return e.At }
func (e ImplementationReviewAgentRejected) OccurredAt() time.Time { return e.At }
func (e ImplementationReviewHumanApproved) OccurredAt() time.Time { return e.At }
func (e ImplementationReviewHumanRejected) OccurredAt() time.Time { return e.At }
