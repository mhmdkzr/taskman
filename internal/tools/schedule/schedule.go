// Package schedule owns the agent.run wire contract that scheduled agent runs
// are published on: RunSubject, the RunRequest payload, and the
// Schedule/ScheduleRecurring helpers that durably record a run through the
// scheduler. The bus itself is consumed by the runtime (the agent package),
// which imports this package rather than the other way round.
package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mhmdkzr/taskman/internal/scheduler"
)

const (
	// RunSubject is the subject scheduled agent runs are published on and the
	// consumer subscribes to.
	RunSubject = "agent.run"
)

// RunRequest is the payload of an agent.run message: everything the consumer
// needs to run one agent. An empty SessionID starts a fresh session.
type RunRequest struct {
	Prompt          string `json:"prompt"`
	SessionID       string `json:"session_id,omitempty"`
	SystemPrompt    string `json:"system_prompt,omitempty"`
	Model           string `json:"model,omitempty"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	MaxSteps        int    `json:"max_steps,omitempty"`
}

// Subject returns the agent.run subject.
func (RunRequest) Subject() string { return RunSubject }

// Schedule durably records req for publication on agent.run at at. The time is
// required: Schedule never defaults it, so callers choose when the run fires.
// It returns the scheduler's message id.
func Schedule(ctx context.Context, sched *scheduler.Scheduler, req RunRequest, at time.Time) (string, error) {
	if at.IsZero() {
		return "", fmt.Errorf("schedule: publish time is required")
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("schedule: %w", err)
	}
	return sched.Schedule(ctx, req.Subject(), payload, at)
}

// ScheduleRecurring durably records req for publication on agent.run every
// interval, starting at now + interval. Each occurrence is a separate run. It
// returns the recurrence id.
func ScheduleRecurring(ctx context.Context, sched *scheduler.Scheduler, req RunRequest, every time.Duration) (string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("schedule recurring: %w", err)
	}
	return sched.ScheduleRecurring(ctx, req.Subject(), payload, every)
}

// MeaningfulOverride trims v and returns it unchanged, or "" when it is empty
// or the sentinel "default". Scheduling agents often emit "default" in optional
// fields to mean "inherit", so consumers treat it like an omitted value.
func MeaningfulOverride(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "default") {
		return ""
	}
	return v
}
