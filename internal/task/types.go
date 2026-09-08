// Package task owns the Task domain type and its file-backed persistence.
// See README.md and notes/design/design.md for the full design.
package task

import "time"

// State is the coarse, top-level lifecycle of a task.
type State string

const (
	StateCreated   State = "created"
	StateStarted   State = "started"
	StateBlocked   State = "blocked"
	StateCompleted State = "completed"
	StateFailed    State = "failed"
)

// StageState is the value of a single stage's status.state field. Not every
// stage reaches every value - see design.md §6's per-stage table.
type StageState string

const (
	StagePending    StageState = "pending"
	StageInProgress StageState = "in_progress"
	StageDone       StageState = "done"
)

// Task is one unit of work, serialized as .tasks/<id>.yaml. Field order here
// matches the schema example in design.md §3.
type Task struct {
	ID            string            `json:"id"                      yaml:"id"`
	State         State             `json:"state"                   yaml:"state"`
	Title         string            `json:"title"                   yaml:"title"`
	Labels        map[string]string `json:"labels,omitempty"        yaml:"labels,omitempty"`
	Definition    string            `json:"definition"              yaml:"definition"`
	Specification string            `json:"specification,omitempty" yaml:"specification,omitempty"`
	DoneWhen      string            `json:"done_when,omitempty"     yaml:"done_when,omitempty"`
	References    []string          `json:"references,omitempty"    yaml:"references,omitempty"`
	Status        Status            `json:"status"                  yaml:"status"`
	Git           Git               `json:"git"                     yaml:"git"`
	Verifications []Verification    `json:"verifications,omitempty" yaml:"verifications,omitempty"`
	Reviews       []Review          `json:"reviews,omitempty"       yaml:"reviews,omitempty"`
	HumanReviews  []HumanReview     `json:"human_reviews,omitempty" yaml:"human_reviews,omitempty"`
	Blocked       *Blocked          `json:"blocked,omitempty"       yaml:"blocked,omitempty"`
	// FailureReason is set by task abandon - the only place a task ever
	// records why it stopped for good.
	FailureReason string `json:"failure_reason,omitempty" yaml:"failure_reason,omitempty"`
}

// Status holds the per-stage progress, one field per design.md §6 stage.
type Status struct {
	Definition     StageStatus `json:"definition"     yaml:"definition"`
	Specification  StageStatus `json:"specification"  yaml:"specification"`
	Implementation StageStatus `json:"implementation" yaml:"implementation"`
	Verification   StageStatus `json:"verification"   yaml:"verification"`
	Review         StageStatus `json:"review"         yaml:"review"`
	Merge          StageStatus `json:"merge"          yaml:"merge"`
}

// StageStatus is one stage's own state, when it completed (if it has), and
// - verification only - how many attempts it took.
type StageStatus struct {
	State       StageState `json:"state"                  yaml:"state"`
	CompletedAt *time.Time `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	Attempts    int        `json:"attempts,omitempty"     yaml:"attempts,omitempty"`
}

// Git holds the task's worktree/branch and its recorded commit, if any.
type Git struct {
	Worktree string     `json:"worktree,omitempty" yaml:"worktree,omitempty"`
	Branch   string     `json:"branch,omitempty"   yaml:"branch,omitempty"`
	Commit   *GitCommit `json:"commit,omitempty"   yaml:"commit,omitempty"`
}

// GitCommit is what task commit reads back from the worktree via git log - see
// design.md §6 "When the commit happens". At is when taskman recorded it
// (not the commit's own author/commit date) - it's what Next uses to tell
// "a fresh commit was already made for the current pass" apart from "the
// commit on file is stale, dispatch drafting a new one".
type GitCommit struct {
	Type    string    `json:"type,omitempty" yaml:"type,omitempty"`
	Message string    `json:"message"        yaml:"message"`
	Hash    string    `json:"hash"           yaml:"hash"`
	At      time.Time `json:"at"             yaml:"at"`
}

// Verification is one task verify call's reported outcome, appended to
// Task.Verifications - never overwritten, so a blocked task's history shows
// exactly which check failed and when.
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

// Passed reports whether every check in the verification succeeded.
func (v Verification) Passed() bool {
	for _, result := range v.Checks {
		if result != CheckOK {
			return false
		}
	}
	return len(v.Checks) > 0
}

// Review is one automated review round's verdict, appended to Task.Reviews.
type Review struct {
	Attempt   int       `json:"attempt"            yaml:"attempt"`
	Approved  bool      `json:"approved"           yaml:"approved"`
	Findings  []Finding `json:"findings,omitempty" yaml:"findings,omitempty"`
	CreatedAt time.Time `json:"created_at"         yaml:"created_at"`
}

// Finding is one automated reviewer's note against a file. Detail is the
// full text of what was found, not a compressed summary - design.md §3.
type Finding struct {
	File   string `json:"file"   yaml:"file"`
	Detail string `json:"detail" yaml:"detail"`
}

// HumanReview is one human decision at the review stage, appended to
// Task.HumanReviews. Deliberately flat - a single text block, not automated
// review's structured per-file findings - design.md §3.
type HumanReview struct {
	Approved bool      `json:"approved"          yaml:"approved"`
	Comment  string    `json:"comment,omitempty" yaml:"comment,omitempty"`
	At       time.Time `json:"at"                yaml:"at"`
}

// Blocked is present only while Task.State is StateBlocked - design.md §6
// "The blocked overlay".
type Blocked struct {
	Stage  string    `json:"stage"  yaml:"stage"`
	Reason string    `json:"reason" yaml:"reason"`
	At     time.Time `json:"at"     yaml:"at"`
}

// Stage names, used in Blocked.Stage and task escalate's --stage flag.
const (
	StageDefinition     = "definition"
	StageSpecification  = "specification"
	StageImplementation = "implementation"
	StageVerification   = "verification"
	StageReview         = "review"
	StageMerge          = "merge"
)

// well-known label keys, validated against a fixed low/medium/high enum
// when present - design.md §3. Every other label key is an unchecked plain
// user tag.
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
