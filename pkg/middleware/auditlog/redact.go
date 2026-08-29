package auditlog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"unicode"
)

// Redactor is a function that redacts sensitive information from an audit event.
type Redactor func(EventAPIAuditLogged) EventAPIAuditLogged

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

// requestHeaderKeys contains request header keys to redact.
var requestHeaderKeys = map[string]struct{}{
	"authorization":      {},
	"proxyauthorization": {},
	"cookie":             {},
	"xapikey":            {},
	"xauthtoken":         {},
	"xcsrftoken":         {},
}

// responseHeaderKeys contains response header keys to redact.
var responseHeaderKeys = map[string]struct{}{
	"setcookie": {},
}

// AuditEvent returns a copy of the event with sensitive data redacted.
func AuditEvent(event EventAPIAuditLogged) EventAPIAuditLogged {
	out := event
	out.Request.Headers = redactHeaders(event.Request.Headers, requestHeaderKeys)
	out.Response.Headers = redactHeaders(event.Response.Headers, responseHeaderKeys)
	out.Request.Query = redactQuery(event.Request.Query)
	out.Request.Body = redactJSONBody(event.Request.Body)
	out.Response.Body = redactJSONBody(event.Response.Body)
	return out
}

// AuditEventInPlace redacts sensitive data from the event in place.
func AuditEventInPlace(event *EventAPIAuditLogged) {
	if event == nil {
		return
	}
	*event = AuditEvent(*event)
}

// redactHeaders returns a copy of headers with sensitive values redacted.
func redactHeaders(headers http.Header, keys map[string]struct{}) http.Header {
	if headers == nil {
		return nil
	}

	out := make(http.Header, len(headers))
	for key, values := range headers {
		copied := append([]string(nil), values...)
		if _, ok := keys[NormalizeKey(key)]; ok {
			for i := range copied {
				copied[i] = Marker
			}
		}
		out[key] = copied
	}
	return out
}

// redactQuery returns the query string with sensitive parameter values redacted.
func redactQuery(raw string) string {
	if raw == "" {
		return ""
	}

	values, err := url.ParseQuery(raw)
	if err != nil {
		return raw
	}
	for key := range values {
		if IsSensitiveKey(key) {
			for i := range values[key] {
				values[key][i] = Marker
			}
		}
	}
	return values.Encode()
}

// redactJSONBody returns the JSON body with sensitive field values redacted.
func redactJSONBody(body []byte) []byte {
	if len(bytes.TrimSpace(body)) == 0 {
		return append([]byte(nil), body...)
	}

	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return append([]byte(nil), body...)
	}

	redactJSONValue(value)
	out, err := json.Marshal(value)
	if err != nil {
		return append([]byte(nil), body...)
	}
	return out
}

// redactJSONValue recursively redacts sensitive values in a JSON tree.
func redactJSONValue(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if IsSensitiveKey(key) {
				typed[key] = Marker
				continue
			}
			redactJSONValue(nested)
		}
	case []any:
		for _, nested := range typed {
			redactJSONValue(nested)
		}
	}
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
