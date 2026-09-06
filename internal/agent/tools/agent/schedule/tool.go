package schedule

import (
	"fmt"
	"time"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/agent/run"
)

const (
	Name        = ""
	Description = ""
)

type Input struct {
	AgentInput   run.Input    `json:"agent_input"         jsonschema:"description=Agent input parameters."`
	ScheduleType ScheduleType `json:"schedule_type"       jsonschema:"description=Schedule type: once, delay or interval."`
	RunAt        string       `json:"run_at,omitempty"    jsonschema:"description=RFC3339 datetime (e.g. 2026-08-14T15:04:05Z) at which the agent runs once. Required unless run_after or run_every is set."`
	RunAfter     string       `json:"run_after,omitempty" jsonschema:"description=Relative time from now (e.g. 30m, 2h, 1d) at which the agent runs once. Equivalent to run_at = now + this duration. Required unless run_at or run_every is set."`
	RunEvery     string       `json:"run_every,omitempty" jsonschema:"description=Interval (e.g. 5m, 1h, 1d) at which the agent repeats indefinitely until cancelled. Required unless run_at or run_after is set."`
}

type ScheduleType string

const (
	ScheduleTypeOnce     ScheduleType = "once"
	ScheduleTypeDelay    ScheduleType = "delay"
	ScheduleTypeInterval ScheduleType = "interval"
)

type Output struct {
	ScheduleID string `json:"schedule_id" jsonschema:"description=The ID of the scheduled task."`
	Scheduled  bool   `json:"scheduled"   jsonschema:"description=Whether the schedule was successfully scheduled."`
	NextRun    string `json:"next_run"    jsonschema:"description=The next scheduled run time (RFC3339)."`
}

func Tool() goai.Tool {
	return tools.Tool(Name, Description, execute)
}

func (i Input) Validate() error {
	if i.ScheduleType == "" {
		return fmt.Errorf("schedule_type is required")
	}

	switch i.ScheduleType {
	case ScheduleTypeInterval:
		if i.RunAt != "" || i.RunAfter != "" {
			return fmt.Errorf("run_at and run_after must be empty for interval schedule")
		}
		if i.RunEvery == "" {
			return fmt.Errorf("run_every is required for interval schedule")
		}
		if _, err := time.ParseDuration(i.RunEvery); err != nil {
			return fmt.Errorf("invalid run_every duration: %w", err)
		}
	case ScheduleTypeOnce:
		if i.RunAfter != "" || i.RunEvery != "" {
			return fmt.Errorf("run_after and run_every must be empty for once schedule")
		}
		if i.RunAt == "" {
			return fmt.Errorf("run_at is required for once schedule")
		}
		if _, err := time.Parse(time.RFC3339, i.RunAt); err != nil {
			return fmt.Errorf("invalid run_at datetime (must be RFC3339): %w", err)
		}
	case ScheduleTypeDelay:
		if i.RunAt != "" || i.RunEvery != "" {
			return fmt.Errorf("run_at and run_every must be empty for delay schedule")
		}
		if i.RunAfter == "" {
			return fmt.Errorf("run_after is required for delay schedule")
		}
		if _, err := time.ParseDuration(i.RunAfter); err != nil {
			return fmt.Errorf("invalid run_after duration: %w", err)
		}
	default:
		return fmt.Errorf("invalid schedule_type: %q", i.ScheduleType)
	}

	return nil
}
