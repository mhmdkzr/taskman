// Package identity owns application user identity mapped from external identity providers.
package identity

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ZitadelSubject is the canonical external identity. It must not be used by other modules.
type ZitadelSubject string

// NewZitadelSubject validates an external identity-provider subject.
func NewZitadelSubject(value string) (ZitadelSubject, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("zitadel subject must not be empty")
	}
	return ZitadelSubject(value), nil
}

// UserID is the application-owned UUIDv7 identity used by all downstream data.
type UserID uuid.UUID

// NewUserID generates a UUIDv7 application user ID.
func NewUserID() (UserID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return UserID{}, fmt.Errorf("generate UUIDv7: %w", err)
	}
	return UserID(id), nil
}

// UUID returns the standard UUID representation.
func (id UserID) UUID() uuid.UUID { return uuid.UUID(id) }

// String returns the canonical UUID string.
func (id UserID) String() string { return id.UUID().String() }
