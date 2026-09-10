// Package taskv1 contains the disposable schema-v1 reader and converter.
package taskv1

import (
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
)

type document struct {
	Task legacyTask `yaml:"task"`
}

type currentDocument struct {
	Task task.Task `yaml:"task"`
}

type legacyTask struct {
	ID            string              `yaml:"id"`
	State         string              `yaml:"state"`
	Title         string              `yaml:"title"`
	Labels        map[string]string   `yaml:"labels,omitempty"`
	Definition    string              `yaml:"definition"`
	Specification string              `yaml:"specification,omitempty"`
	DoneWhen      string              `yaml:"done_when,omitempty"`
	References    []string            `yaml:"references,omitempty"`
	Status        legacyStatus        `yaml:"status"`
	Git           task.Git            `yaml:"git"`
	Verifications []task.Verification `yaml:"verifications,omitempty"`
	Reviews       []task.Review       `yaml:"reviews,omitempty"`
	HumanReviews  []task.HumanReview  `yaml:"human_reviews,omitempty"`
	Blocked       *legacyBlocked      `yaml:"blocked,omitempty"`
	FailureReason string              `yaml:"failure_reason,omitempty"`
	AutoApprove   bool                `yaml:"auto_approve,omitempty"`
}

type legacyStatus struct {
	Definition     legacyStageStatus `yaml:"definition"`
	Specification  legacyStageStatus `yaml:"specification"`
	Implementation legacyStageStatus `yaml:"implementation"`
	Verification   legacyStageStatus `yaml:"verification"`
	Review         legacyStageStatus `yaml:"review"`
	Merge          legacyStageStatus `yaml:"merge"`
}

type legacyStageStatus struct {
	State       string     `yaml:"state"`
	CompletedAt *time.Time `yaml:"completed_at,omitempty"`
	Attempts    int        `yaml:"attempts,omitempty"`
}

type legacyBlocked struct {
	Stage  string    `yaml:"stage"`
	Reason string    `yaml:"reason"`
	At     time.Time `yaml:"at"`
}
