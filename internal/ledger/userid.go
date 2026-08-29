// Package ledger holds TigerBeetle conventions shared by future ledger slices.
package ledger

import (
	tigerbeetle "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/mhmdkzr/app/internal/identity"
)

// UserIDToUInt128 reinterprets the UUIDv7's raw 16 bytes as a TigerBeetle UInt128.
func UserIDToUInt128(userID identity.UserID) tigerbeetle.Uint128 {
	return tigerbeetle.BytesToUint128(userID.UUID())
}
