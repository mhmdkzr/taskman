package todo

import "errors"

type TodoStatus string

const (
	TodoPending    TodoStatus = "pending"
	TodoInProgress TodoStatus = "in_progress"
	TodoCompleted  TodoStatus = "completed"
	TodoCancelled  TodoStatus = "cancelled"
)

type TodoPriority string

const (
	TodoHigh   TodoPriority = "high"
	TodoMedium TodoPriority = "medium"
	TodoLow    TodoPriority = "low"
)

var ErrInvalidTodo = errors.New("invalid todo")

type Todo struct {
	Content  string       `json:"content"`
	Status   TodoStatus   `json:"status"`
	Priority TodoPriority `json:"priority"`
}
