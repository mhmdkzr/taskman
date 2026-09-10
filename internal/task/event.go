package task

import "time"

// EventKind identifies a fact reported to the workflow.
type EventKind string

const (
	EventSpecificationSubmitted  EventKind = "specification_submitted"
	EventSpecificationApproved   EventKind = "specification_approved"
	EventSpecificationRejected   EventKind = "specification_rejected"
	EventImplementationCompleted EventKind = "implementation_completed"
	EventVerificationReported    EventKind = "verification_reported"
	EventAutomatedReviewRecorded EventKind = "automated_review_recorded"
	EventCommitRecorded          EventKind = "commit_recorded"
	EventHumanReviewApproved     EventKind = "human_review_approved"
	EventHumanReviewRejected     EventKind = "human_review_rejected"
	EventMergeCompleted          EventKind = "merge_completed"
	EventEscalated               EventKind = "escalated"
	EventAbandoned               EventKind = "abandoned"
)

// Event is a closed set of typed facts accepted by Apply.
type Event interface{ eventKind() EventKind }

type SpecificationSubmitted struct{ Specification, DoneWhen string }

func (SpecificationSubmitted) eventKind() EventKind { return EventSpecificationSubmitted }

type SpecificationApproved struct {
	Comment string
	At      time.Time
}

func (SpecificationApproved) eventKind() EventKind { return EventSpecificationApproved }

type SpecificationRejected struct {
	Reason string
	At     time.Time
}

func (SpecificationRejected) eventKind() EventKind { return EventSpecificationRejected }

type ImplementationCompleted struct{}

func (ImplementationCompleted) eventKind() EventKind { return EventImplementationCompleted }

type VerificationReported struct{ Verification Verification }

func (VerificationReported) eventKind() EventKind { return EventVerificationReported }

type AutomatedReviewRecorded struct {
	Approved bool
	Findings []Finding
	At       time.Time
}

func (AutomatedReviewRecorded) eventKind() EventKind { return EventAutomatedReviewRecorded }

type CommitRecorded struct{ Commit GitCommit }

func (CommitRecorded) eventKind() EventKind { return EventCommitRecorded }

type HumanReviewApproved struct {
	Comment string
	At      time.Time
}

func (HumanReviewApproved) eventKind() EventKind { return EventHumanReviewApproved }

type HumanReviewRejected struct {
	Reason string
	At     time.Time
}

func (HumanReviewRejected) eventKind() EventKind { return EventHumanReviewRejected }

type MergeCompleted struct{ CommitOverride string }

func (MergeCompleted) eventKind() EventKind { return EventMergeCompleted }

type Escalated struct {
	Stage, Reason string
	At            time.Time
}

func (Escalated) eventKind() EventKind { return EventEscalated }

type Abandoned struct{ Reason string }

func (Abandoned) eventKind() EventKind { return EventAbandoned }
