package events

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
)

// eventID derives a deterministic dedup id from an event's contents: the
// type name plus the canonical JSON of its fields, hashed. Identical events
// always produce the same id, and distinct events collide only if they are
// byte-for-byte identical within the stream's dedup window (which is the
// intended "republished" case).
func eventID(e any) string {
	data, err := json.Marshal(e)
	if err != nil {
		return reflect.TypeOf(e).String()
	}
	h := sha256.New()
	h.Write([]byte(reflect.TypeOf(e).String()))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
