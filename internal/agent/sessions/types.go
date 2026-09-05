// Package sessions persists and runs agent sessions.
package sessions

import (
	"uuid"
)

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
