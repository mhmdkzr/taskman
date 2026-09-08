// Package task owns the Task domain type and its persistence (tasks,
// tasks_labels, tasks_sessions, task_review_results). See README.md.
package task

import (
	"errors"
	"uuid"
)

var ErrInvalidLevel = errors.New("invalid level")

type Task struct {
	ID              uuid.UUID
	Definition      string
	Specification   string
	State           TaskState
	Labels          []string
	Importance      Level
	Urgency         Level
	Complexity      Level
	Effort          Level
	Risk            Level
	Autonomy        Level
	Model           string
	ReasoningEffort string
	CommitHash      string
	Branch          string
	FailureReason   string
	// PipelineStep is opaque to this package - it's the autonomous
	// pipeline's own progress marker (see internal/pipeline.PipelineStep),
	// stored here only because a task's pipeline progress is task state.
	// Empty for a task not created by the pipeline.
	PipelineStep string
}

type Level int

const (
	LevelVeryLow Level = iota + 1
	LevelLow
	LevelMedium
	LevelHigh
	LevelVeryHigh
)

type TaskState string

const (
	TaskStateCreated   TaskState = "created"
	TaskStateStarted   TaskState = "started"
	TaskStateCompleted TaskState = "completed"
	TaskStateCancelled TaskState = "cancelled"
	TaskStateBlocked   TaskState = "blocked"
	TaskStateFailed    TaskState = "failed"
)

type TaskFilter struct {
	IDs              []uuid.UUID `json:"ids,omitempty"`
	State            []TaskState `json:"state,omitempty"`
	Labels           []string    `json:"labels,omitempty"`
	Importance       []Level     `json:"importance,omitempty"`
	Urgency          []Level     `json:"urgency,omitempty"`
	Complexity       []Level     `json:"complexity,omitempty"`
	Effort           []Level     `json:"effort,omitempty"`
	Risk             []Level     `json:"risk,omitempty"`
	Autonomy         []Level     `json:"autonomy,omitempty"`
	Model            []string    `json:"model,omitempty"`
	ReasoningEfforts []string    `json:"reasoning_efforts,omitempty"`
	CommitHashes     []string    `json:"commit_hashes,omitempty"`
	Branches         []string    `json:"branches,omitempty"`
}

// ReviewFinding is one point raised by a review attempt against a task.
type ReviewFinding struct {
	File    string `json:"file"`
	Summary string `json:"summary"`
}

// ReviewResult is one row of task_review_results: a single review attempt's
// outcome, persisted the moment the review finishes so a crash afterward
// never loses it (see internal/pipeline).
type ReviewResult struct {
	ID        uuid.UUID
	TaskID    uuid.UUID
	SessionID uuid.UUID
	Attempt   int
	Approved  bool
	Findings  []ReviewFinding
}

func (l Level) Validate() error {
	switch l {
	case LevelVeryLow, LevelLow, LevelMedium, LevelHigh, LevelVeryHigh:
		return nil
	default:
		return ErrInvalidLevel
	}
}
