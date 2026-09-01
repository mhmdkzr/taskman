package task

import (
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/usage"
)

type Task struct {
	ID              uuid.UUID        `json:"id"`
	Title           string           `json:"title"`
	Status          TaskStatus       `json:"status"`
	TaskType        string           `json:"task_type"`
	Tags            []string         `json:"tags,omitempty"`
	Dependencies    []string         `json:"dependencies,omitempty"`
	What            string           `json:"what"`
	Where           []string         `json:"where"`
	Why             string           `json:"why"`
	How             string           `json:"how"`
	Invariants      []string         `json:"invariants,omitempty"`
	Urgency         string           `json:"urgency"`
	Importance      string           `json:"importance"`
	Risk            string           `json:"risk"` // blast radius if execution goes wrong
	Packages        []string         `json:"packages"`
	CompletedWhen   []string         `json:"completed_when"`
	CommitType      string           `json:"commit_type,omitzero"`
	CommitMessage   string           `json:"commit_message,omitzero"`
	CommitHash      string           `json:"commit_hash,omitzero"`
	Model           string           `json:"model,omitzero"`
	Variant         string           `json:"variant,omitzero"`
	SessionID       uuid.UUID        `json:"session_id,omitzero"`
	CreatedAt       time.Time        `json:"created_at,omitzero"`
	StartedAt       time.Time        `json:"started_at,omitzero"`
	CompletedAt     time.Time        `json:"completed_at,omitzero"`
	ReviewedAt      time.Time        `json:"reviewed_at,omitzero"`
	ReviewSessionID uuid.UUID        `json:"review_session_id,omitzero"`
	TokenUsage      usage.TokenUsage `json:"token_usage,omitzero"`
}

type TaskStatus string

const (
	TaskStatusCreated   TaskStatus = "created"
	TaskStatusStarted   TaskStatus = "started"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusReviewed  TaskStatus = "reviewed"
)

var (
	validCommitTypes = []string{
		"feat",
		"fix",
		"refactor",
		"chore",
		"test",
		"docs",
		"style",
		"perf",
		"revert",
		"build",
		"ci",
		"misc",
	}
	validVariants   = []string{"low", "medium", "high", "xhigh", "max", "default"}
	validTaskTypes  = []string{"bug", "docs", "feature", "refactor", "test"}
	validUrgency    = []string{"low", "medium", "high"}
	validImportance = []string{"low", "medium", "high"}
	validRisk       = []string{"low", "medium", "high"}
)

func (t Task) Validate() error {
	if !slices.Contains(validTaskTypes, t.TaskType) {
		return fmt.Errorf("invalid task type: %s", t.TaskType)
	}
	if !slices.Contains(validUrgency, t.Urgency) {
		return fmt.Errorf("invalid urgency: %s", t.Urgency)
	}
	if !slices.Contains(validImportance, t.Importance) {
		return fmt.Errorf("invalid importance: %s", t.Importance)
	}
	if !slices.Contains(validRisk, t.Risk) {
		return fmt.Errorf("invalid risk: %s", t.Risk)
	}
	if !slices.Contains(validCommitTypes, t.CommitType) {
		return fmt.Errorf("invalid commit type: %s", t.CommitType)
	}
	if !slices.Contains(validVariants, t.Variant) {
		return fmt.Errorf("invalid variant: %s", t.Variant)
	}
	if t.ID == uuid.Nil() {
		return fmt.Errorf("invalid id: %s", t.ID)
	}
	if t.Title == "" {
		return fmt.Errorf("invalid title: %s", t.Title)
	}
	if t.What == "" {
		return fmt.Errorf("invalid what: %s", t.What)
	}
	if t.Why == "" {
		return fmt.Errorf("invalid why: %s", t.Why)
	}
	if t.How == "" {
		return fmt.Errorf("invalid how: %s", t.How)
	}
	if t.Packages == nil {
		return fmt.Errorf("invalid packages: %v", t.Packages)
	}
	if t.CompletedWhen == nil {
		return fmt.Errorf("invalid completed_when: %v", t.CompletedWhen)
	}

	return nil
}
