package task

import (
	"errors"
	"uuid"
)

var ErrInvalidLevel = errors.New("invalid level")

type Task struct {
	ID            uuid.UUID
	Definition    string
	Specification string
	State         TaskState
	Labels        []string
	Importance    Level
	Urgency       Level
	Complexity    Level
	Effort        Level
	Risk          Level
	Autonomy      Level
	Model         string
	CommitHash    string
	Branch        string
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
	IDs          []uuid.UUID `json:"ids,omitempty"`
	State        []TaskState `json:"state,omitempty"`
	Labels       []string    `json:"labels,omitempty"`
	Importance   []Level     `json:"importance,omitempty"`
	Urgency      []Level     `json:"urgency,omitempty"`
	Complexity   []Level     `json:"complexity,omitempty"`
	Effort       []Level     `json:"effort,omitempty"`
	Risk         []Level     `json:"risk,omitempty"`
	Autonomy     []Level     `json:"autonomy,omitempty"`
	Model        []string    `json:"model,omitempty"`
	CommitHashes []string    `json:"commit_hashes,omitempty"`
	Branches     []string    `json:"branches,omitempty"`
}

func (l Level) Validate() error {
	switch l {
	case LevelVeryLow, LevelLow, LevelMedium, LevelHigh, LevelVeryHigh:
		return nil
	default:
		return ErrInvalidLevel
	}
}
