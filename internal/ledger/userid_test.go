package ledger

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mhmdkzr/app/internal/identity"
)

func TestUserIDToUInt128PreservesRawUUIDBytes(t *testing.T) {
	t.Parallel()
	id := identity.UserID(uuid.MustParse("019c1548-3c8d-7000-8000-000000000001"))
	if got := UserIDToUInt128(id).Bytes(); got != id.UUID() {
		t.Fatalf("UInt128 bytes = %x, want %x", got, id.UUID())
	}
}
