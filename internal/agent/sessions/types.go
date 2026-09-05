// Package sessions persists and runs agent sessions.
package sessions

import (
	"errors"
	"uuid"
)

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

// SessionID is the application-owned UUIDv7 identity of an agent session.
type SessionID uuid.UUID

// UUID returns the standard UUID representation.
func (id SessionID) UUID() uuid.UUID {
	return uuid.UUID(id)
}

// String returns the canonical UUID string.
func (id SessionID) String() string {
	return id.UUID().String()
}

// ModelID is the UUID identity of the models row a session runs on.
type ModelID uuid.UUID

// UUID returns the standard UUID representation.
func (id ModelID) UUID() uuid.UUID {
	return uuid.UUID(id)
}

// String returns the canonical UUID string.
func (id ModelID) String() string {
	return id.UUID().String()
}

// AgentID is the UUID identity of the agents row a session runs as.
type AgentID uuid.UUID

// UUID returns the standard UUID representation.
func (id AgentID) UUID() uuid.UUID {
	return uuid.UUID(id)
}

// String returns the canonical UUID string.
func (id AgentID) String() string {
	return id.UUID().String()
}
