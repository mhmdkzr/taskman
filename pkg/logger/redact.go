package logger

import (
	"strings"
	"unicode"
)

// Marker is the replacement string for redacted sensitive data.
const Marker = "[REDACTED]"

// SensitiveKeys contains normalized keys whose values should be redacted.
// Keys are normalized via NormalizeKey before lookup.
var SensitiveKeys = map[string]struct{}{
	"password":           {},
	"secret":             {},
	"clientsecret":       {},
	"token":              {},
	"accesstoken":        {},
	"refreshtoken":       {},
	"idtoken":            {},
	"assertion":          {},
	"clientassertion":    {},
	"code":               {},
	"state":              {},
	"nonce":              {},
	"codeverifier":       {},
	"codechallenge":      {},
	"apikey":             {},
	"privatekey":         {},
	"mnemonic":           {},
	"passphrase":         {},
	"authorization":      {},
	"proxyauthorization": {},
	"cookie":             {},
	"xapikey":            {},
	"xauthtoken":         {},
	"xcsrftoken":         {},
	"setcookie":          {},
}

// IsSensitiveKey returns true if the key should be redacted.
func IsSensitiveKey(key string) bool {
	_, ok := SensitiveKeys[NormalizeKey(key)]
	return ok
}

// NormalizeKey normalizes a header key to a lowercase string without separators.
func NormalizeKey(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for _, r := range key {
		if r == '_' || r == '-' || unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
