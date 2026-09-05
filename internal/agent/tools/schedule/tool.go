package schedule

import (
	"fmt"
	"time"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = ""
	description = ""
)

type input struct { // TODO: add jsonschema tags
	ScheduleType ScheduleType   `json:"schedule_type"`
	RunAt        *time.Time     `json:"run_at,omitempty"`
	RunEvery     *time.Duration `json:"run_every,omitempty"`
}

type ScheduleType string

const (
	ScheduleTypeOnce     ScheduleType = "once"
	ScheduleTypeInterval ScheduleType = "interval"
)

type output struct{}

func Tool() goai.Tool {
	return tools.Tool(Name, description, execute)
}

func (i input) Validate() error {
	if i.ScheduleType == "" {
		return fmt.Errorf("schedule_type is required")
	}
	if i.ScheduleType == ScheduleTypeInterval {
		if i.RunEvery == nil {
			return fmt.Errorf("run_every is required for interval schedule")
		}
	}
	if i.ScheduleType == ScheduleTypeOnce {
		if i.RunAt == nil {
			return fmt.Errorf("run_at is required for once schedule")
		}
	}
	return nil
}
