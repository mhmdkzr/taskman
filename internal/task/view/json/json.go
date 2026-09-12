// Package json renders a task as a machine-readable JSON document. Unlike
// the raw task serialization (which stores the state history), the rendered
// document also carries the two derived fields an agent needs to drive the
// task: its current state and its next instruction, matching taskman's
// agent-facing skill (SKILL.md). It is presentation, never persistence - a
// Document is always projected from an already-loaded task.
package json

import (
	"github.com/mhmdkzr/taskman/internal/task"
)

// Document is a task rendered for machine consumption: the full task plus
// its derived state and instruction.
type Document struct {
	Task        task.Task        `json:"task"`
	State       task.TaskState   `json:"state"`
	Instruction task.Instruction `json:"instruction"`
}

// FromTask projects t into a Document, deriving state and instruction from
// the task's current position in the workflow.
func FromTask(t task.Task) Document {
	return Document{
		Task:        t,
		State:       t.State(),
		Instruction: t.Instruction(),
	}
}
