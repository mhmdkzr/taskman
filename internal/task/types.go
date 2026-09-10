// Package task owns taskman's pure task model and compiled workflow.
package task

import "time"

// State is the task's single authoritative position in the workflow.
type State string

const (
	StateSpecify                    State = "specify"
	StateSpecificationReview        State = "specification_review"
	StateImplement                  State = "implement"
	StateVerify                     State = "verify"
	StateFixVerificationFailure     State = "fix_verification_failure"
	StateFixAutomatedReviewFindings State = "fix_automated_review_findings"
	StateAutomatedReview            State = "automated_review"
	StateCommit                     State = "commit"
	StateHumanReview                State = "human_review"
	StateFixHumanReviewFindings     State = "fix_human_review_findings"
	StateMerge                      State = "merge"
	StateBlocked                    State = "blocked"
	StateCompleted                  State = "completed"
	StateAbandoned                  State = "abandoned"
)

// Terminal reports whether no further workflow event can advance s.
func (s State) Terminal() bool { return s == StateCompleted || s == StateAbandoned }

// AutoApproveMoot reports whether a task in state s has already passed the
// point where AutoApprove is consulted (EventCommitRecorded's routing) with
// no way back to it. Setting AutoApprove while in one of these states cannot
// change the task's outcome.
func (s State) AutoApproveMoot() bool {
	switch s {
	case StateHumanReview, StateMerge, StateBlocked, StateCompleted, StateAbandoned:
		return true
	default:
		return false
	}
}

// Task is one unit of work, serialized as .tasks/<id>.yaml.
type Task struct {
	ID                   string                `json:"id"                              yaml:"id"`
	State                State                 `json:"state"                           yaml:"state"`
	Title                string                `json:"title"                           yaml:"title"`
	Labels               map[string]string     `json:"labels,omitempty"                yaml:"labels,omitempty"`
	Definition           string                `json:"definition"                      yaml:"definition"`
	Specification        string                `json:"specification,omitempty"         yaml:"specification,omitempty"`
	DoneWhen             string                `json:"done_when,omitempty"             yaml:"done_when,omitempty"`
	References           []string              `json:"references,omitempty"            yaml:"references,omitempty"`
	Git                  Git                   `json:"git"                             yaml:"git"`
	Verifications        []Verification        `json:"verifications,omitempty"         yaml:"verifications,omitempty"`
	Reviews              []Review              `json:"reviews,omitempty"               yaml:"reviews,omitempty"`
	SpecificationReviews []SpecificationReview `json:"specification_reviews,omitempty" yaml:"specification_reviews,omitempty"`
	HumanReviews         []HumanReview         `json:"human_reviews,omitempty"         yaml:"human_reviews,omitempty"`
	Blocked              *Blocked              `json:"blocked,omitempty"               yaml:"blocked,omitempty"`
	FailureReason        string                `json:"failure_reason,omitempty"        yaml:"failure_reason,omitempty"`
	AutoApprove          bool                  `json:"auto_approve,omitempty"          yaml:"auto_approve,omitempty"`
}

// Git holds the task's worktree/branch and its recorded commit, if any.
type Git struct {
	Worktree string     `json:"worktree,omitempty" yaml:"worktree,omitempty"`
	Branch   string     `json:"branch,omitempty"   yaml:"branch,omitempty"`
	Trunk    bool       `json:"trunk,omitempty"    yaml:"trunk,omitempty"`
	Commit   *GitCommit `json:"commit,omitempty"   yaml:"commit,omitempty"`
}

// GitCommit is a commit observed by taskman through Git.
type GitCommit struct {
	Type    string    `json:"type,omitempty" yaml:"type,omitempty"`
	Message string    `json:"message"        yaml:"message"`
	Hash    string    `json:"hash"           yaml:"hash"`
	At      time.Time `json:"at"             yaml:"at"`
}

// Verification is one reported build-check attempt.
type Verification struct {
	Checks    map[string]CheckResult `json:"checks"           yaml:"checks"`
	Output    string                 `json:"output,omitempty" yaml:"output,omitempty"`
	CreatedAt time.Time              `json:"created_at"       yaml:"created_at"`
}

// CheckResult is one named check's outcome within a Verification.
type CheckResult string

const (
	CheckOK    CheckResult = "ok"
	CheckError CheckResult = "error"
)

// Passed reports whether every reported check succeeded.
func (v Verification) Passed() bool {
	for _, result := range v.Checks {
		if result != CheckOK {
			return false
		}
	}
	return len(v.Checks) > 0
}

// Review is one automated review round's verdict.
type Review struct {
	Attempt   int       `json:"attempt"            yaml:"attempt"`
	Approved  bool      `json:"approved"           yaml:"approved"`
	Findings  []Finding `json:"findings,omitempty" yaml:"findings,omitempty"`
	CreatedAt time.Time `json:"created_at"         yaml:"created_at"`
}

// Finding is one automated reviewer's note against a file.
type Finding struct {
	File   string `json:"file"   yaml:"file"`
	Detail string `json:"detail" yaml:"detail"`
}

// HumanReview is one human decision at the human-review state.
type HumanReview struct {
	Approved bool      `json:"approved"          yaml:"approved"`
	Comment  string    `json:"comment,omitempty" yaml:"comment,omitempty"`
	At       time.Time `json:"at"                yaml:"at"`
}

// SpecificationReview is one human decision on a drafted specification.
type SpecificationReview struct {
	Approved bool      `json:"approved"          yaml:"approved"`
	Comment  string    `json:"comment,omitempty" yaml:"comment,omitempty"`
	At       time.Time `json:"at"                yaml:"at"`
}

// Blocked describes a suspended task and the state to resume at.
type Blocked struct {
	ResumeState State     `json:"resume_state" yaml:"resume_state"`
	Stage       string    `json:"stage"        yaml:"stage"`
	Reason      string    `json:"reason"       yaml:"reason"`
	At          time.Time `json:"at"           yaml:"at"`
}

// Stage names remain part of the escalate command's caller-facing vocabulary.
const (
	StageDefinition     = "definition"
	StageSpecification  = "specification"
	StageImplementation = "implementation"
	StageVerification   = "verification"
	StageReview         = "review"
	StageMerge          = "merge"
)

const (
	LabelPriority   = "priority"
	LabelComplexity = "complexity"
	LabelAutonomy   = "autonomy"
)

const (
	LevelLow    = "low"
	LevelMedium = "medium"
	LevelHigh   = "high"
)
