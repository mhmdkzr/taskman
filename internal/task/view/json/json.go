// Package json renders a task as a machine-readable JSON document. Unlike
// the raw task serialization (which stores the state history), the rendered
// document also carries the derived fields an agent needs to drive the
// task: its current state, its next instruction, the rendered guidance
// message, and the command(s) that would currently report an outcome -
// matching taskman's agent-facing skill (SKILL.md). It is presentation,
// never persistence - a Document is always projected from an
// already-loaded task.
package json

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/view/instructions"
)

// Document is a task rendered for machine consumption: the full task plus
// its derived state, instruction, and per-state guidance.
type Document struct {
	Task        task.Task        `json:"task"`
	State       task.TaskState   `json:"state"`
	Instruction task.Instruction `json:"instruction"`
	Message     string           `json:"message"`
	Commands    []string         `json:"commands,omitempty"`
}

// FromTask projects t into a Document, deriving state, instruction, and
// guidance from the task's current position in the workflow.
func FromTask(t task.Task) (Document, error) {
	instructions, err := instructions.Project(t)
	if err != nil {
		return Document{}, fmt.Errorf("project instructions: %w", err)
	}
	return Document{
		Task:        t,
		State:       instructions.State,
		Instruction: task.Instruction{State: instructions.State, Action: instructions.Action},
		Message:     instructions.Message,
		Commands:    instructions.Commands,
	}, nil
}
