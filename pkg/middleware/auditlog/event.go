package auditlog

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mhmdkzr/app/pkg/middleware"
)

// EventAPIAuditLogged contains the full audit trail for a single API request.
type EventAPIAuditLogged struct {
	RequestID string                  `json:"request_id"`
	Timestamp time.Time               `json:"timestamp"`
	Duration  time.Duration           `json:"duration"`
	Request   middleware.RequestData  `json:"request"`
	Response  middleware.ResponseData `json:"response"`
}

// eventJSON is the JSON-serializable representation of an audit event.
type eventJSON struct {
	RequestID string        `json:"request_id"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
	Request   requestJSON   `json:"request"`
	Response  responseJSON  `json:"response"`
}

// requestJSON is the JSON-serializable representation of request data.
type requestJSON struct {
	ID         string          `json:"id"`
	Method     string          `json:"method"`
	Path       string          `json:"path"`
	Query      string          `json:"query"`
	RemoteAddr string          `json:"remote_addr"`
	UserAgent  string          `json:"user_agent"`
	Proto      string          `json:"proto"`
	Headers    http.Header     `json:"headers"`
	Body       json.RawMessage `json:"body,omitempty"`
	Truncated  bool            `json:"truncated,omitempty"`
}

// responseJSON is the JSON-serializable representation of response data.
type responseJSON struct {
	StatusCode int             `json:"status_code"`
	Headers    http.Header     `json:"headers"`
	Body       json.RawMessage `json:"body,omitempty"`
	Size       int64           `json:"size"`
	Truncated  bool            `json:"truncated,omitempty"`
}

// MarshalJSON implements json.Marshaler for APIAuditLogged.
func (e EventAPIAuditLogged) MarshalJSON() ([]byte, error) {
	return marshalEvent(e)
}

// marshalEvent marshals an audit event to JSON.
func marshalEvent(e EventAPIAuditLogged) ([]byte, error) {
	return json.Marshal(eventJSON{
		RequestID: e.RequestID,
		Timestamp: e.Timestamp,
		Duration:  e.Duration,
		Request: requestJSON{
			ID:         e.Request.ID,
			Method:     e.Request.Method,
			Path:       e.Request.Path,
			Query:      e.Request.Query,
			RemoteAddr: e.Request.RemoteAddr,
			UserAgent:  e.Request.UserAgent,
			Proto:      e.Request.Proto,
			Headers:    e.Request.Headers,
			Body:       bodyToRawMessage(e.Request.Body),
			Truncated:  e.Request.Truncated,
		},
		Response: responseJSON{
			StatusCode: e.Response.StatusCode,
			Headers:    e.Response.Headers,
			Body:       bodyToRawMessage(e.Response.Body),
			Size:       e.Response.Size,
			Truncated:  e.Response.Truncated,
		},
	})
}

// UnmarshalJSON implements json.Unmarshaler for APIAuditLogged.
func (e *EventAPIAuditLogged) UnmarshalJSON(data []byte) error {
	var decoded eventJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*e = EventAPIAuditLogged{
		RequestID: decoded.RequestID,
		Timestamp: decoded.Timestamp,
		Duration:  decoded.Duration,
		Request: middleware.RequestData{
			ID:         decoded.Request.ID,
			Method:     decoded.Request.Method,
			Path:       decoded.Request.Path,
			Query:      decoded.Request.Query,
			RemoteAddr: decoded.Request.RemoteAddr,
			UserAgent:  decoded.Request.UserAgent,
			Proto:      decoded.Request.Proto,
			Headers:    decoded.Request.Headers,
			Body:       rawMessageToBody(decoded.Request.Body),
			Truncated:  decoded.Request.Truncated,
		},
		Response: middleware.ResponseData{
			StatusCode: decoded.Response.StatusCode,
			Headers:    decoded.Response.Headers,
			Body:       rawMessageToBody(decoded.Response.Body),
			Size:       decoded.Response.Size,
			Truncated:  decoded.Response.Truncated,
		},
	}
	return nil
}

// MsgID returns the unique message ID for the audit event.
func (e EventAPIAuditLogged) MsgID() string {
	return e.RequestID
}

// bodyToRawMessage converts a byte slice to a json.RawMessage.
func bodyToRawMessage(body []byte) json.RawMessage {
	if len(body) == 0 {
		return nil
	}
	if json.Valid(body) {
		return json.RawMessage(body)
	}
	s := string(body)
	b, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

// rawMessageToBody converts a json.RawMessage back to a byte slice.
func rawMessageToBody(msg json.RawMessage) []byte {
	if len(msg) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(msg, &s); err == nil {
		return []byte(s)
	}
	return []byte(msg)
}
