package components

import (
	"github.com/mhmdkzr/taskman/internal/task"
)

// KanbanCol is one status column of the board: a heading plus its task cards.
type KanbanCol struct {
	Status string
	Label  string
	Tasks  []TaskRow
}

// TaskRow is one card on the board: a task plus its live pipeline phase.
type TaskRow struct {
	task.Task

	Phase string
}

// TaskDetail is the data the detail drawer renders for one task.
type TaskDetail struct {
	Task     task.Task
	Phase    string
	Sessions []SessionActivity
}

// SessionActivity is the rendered transcript of one session (execution or
// review), oldest turn first.
type SessionActivity struct {
	Label string // "execution" | "review"
	Turns []ActivityTurn
}

// ActivityTurn is one turn of a session: its prompt and the steps that follow.
type ActivityTurn struct {
	Prompt string
	Steps  []ActivityStep
}

// ActivityStep is one tool-loop step within a turn.
type ActivityStep struct {
	Reasoning string
	Text      string
	Tools     []ActivityTool
}

// ActivityTool is one tool call in a step with its result.
type ActivityTool struct {
	Name   string
	Input  string
	Output string
	Error  string
}
