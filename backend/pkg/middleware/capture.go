package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// contextKey is the type for context keys in the middleware package.
type contextKey int

const (
	contextKeyRequestID contextKey = iota
	contextKeyStartTime
	contextKeyClientIP
	contextKeyRecorder
)

// WithRequestID stores a request ID in the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKeyRequestID, id)
}

// RequestIDFromContext returns the request ID from the context.
func RequestIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(contextKeyRequestID).(string)
	if !ok {
		return ""
	}
	return id
}

// WithStartTime stores a start time in the context.
func WithStartTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, contextKeyStartTime, t)
}

// StartTimeFromContext returns the start time from the context.
func StartTimeFromContext(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(contextKeyStartTime).(time.Time)
	return t, ok
}

// WithClientIP stores a client IP in the context.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, contextKeyClientIP, ip)
}

// ClientIPFromContext returns the client IP from the context.
func ClientIPFromContext(ctx context.Context) string {
	ip, ok := ctx.Value(contextKeyClientIP).(string)
	if !ok {
		return ""
	}
	return ip
}

// WithRecorder attaches a response recorder to the request context.
func WithRecorder(r *http.Request, rec *ResponseRecorder) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKeyRecorder, rec))
}

// RecorderFromContext returns the response recorder from the request context.
func RecorderFromContext(r *http.Request) (*ResponseRecorder, bool) {
	rec, ok := r.Context().Value(contextKeyRecorder).(*ResponseRecorder)
	return rec, ok
}

// ResponseRecorder captures the response status code, headers, and body.
type ResponseRecorder struct {
	http.ResponseWriter

	StatusCode   int
	Body         *bytes.Buffer
	BytesWritten int64
	Truncated    bool
	wroteHeader  bool
	bodyLimit    int64
}

// DefaultCaptureBodyLimit is the default maximum number of bytes to capture from request/response bodies.
const DefaultCaptureBodyLimit int64 = 1 << 20

// NewResponseRecorder creates a ResponseRecorder with the default body capture limit.
func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return NewResponseRecorderWithLimit(w, DefaultCaptureBodyLimit)
}

// NewResponseRecorderWithLimit creates a ResponseRecorder with the given body capture limit.
func NewResponseRecorderWithLimit(w http.ResponseWriter, bodyLimit int64) *ResponseRecorder {
	if bodyLimit <= 0 {
		bodyLimit = DefaultCaptureBodyLimit
	}
	return &ResponseRecorder{
		ResponseWriter: w,
		Body:           &bytes.Buffer{},
		StatusCode:     http.StatusOK,
		bodyLimit:      bodyLimit,
	}
}

// WriteHeader captures the status code and delegates to the wrapped ResponseWriter.
func (r *ResponseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.StatusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Write captures the written bytes and delegates to the wrapped ResponseWriter.
func (r *ResponseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.BytesWritten += int64(n)
	if r.Body != nil && int64(r.Body.Len()) < r.bodyLimit {
		remaining := r.bodyLimit - int64(r.Body.Len())
		if int64(len(b)) > remaining {
			r.Body.Write(b[:remaining])
			r.Truncated = true
		} else {
			r.Body.Write(b)
		}
	} else if len(b) > 0 {
		r.Truncated = true
	}
	if err != nil {
		return n, fmt.Errorf("write captured response: %w", err)
	}
	return n, nil
}

// RequestData captures the details of an HTTP request.
type RequestData struct {
	ID         string      `json:"id"`
	Method     string      `json:"method"`
	Path       string      `json:"path"`
	Query      string      `json:"query"`
	RemoteAddr string      `json:"remote_addr"`
	UserAgent  string      `json:"user_agent"`
	Proto      string      `json:"proto"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body,omitempty"`
	Truncated  bool        `json:"truncated,omitempty"`
}

// MarshalJSON implements json.Marshaler for RequestData.
func (d RequestData) MarshalJSON() ([]byte, error) {
	type requestData struct {
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

	return json.Marshal(requestData{
		ID:         d.ID,
		Method:     d.Method,
		Path:       d.Path,
		Query:      d.Query,
		RemoteAddr: d.RemoteAddr,
		UserAgent:  d.UserAgent,
		Proto:      d.Proto,
		Headers:    d.Headers,
		Body:       bodyToRawMessage(d.Body),
		Truncated:  d.Truncated,
	})
}

// UnmarshalJSON implements json.Unmarshaler for RequestData.
func (d *RequestData) UnmarshalJSON(data []byte) error {
	type requestData struct {
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

	var decoded requestData
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*d = RequestData{
		ID:         decoded.ID,
		Method:     decoded.Method,
		Path:       decoded.Path,
		Query:      decoded.Query,
		RemoteAddr: decoded.RemoteAddr,
		UserAgent:  decoded.UserAgent,
		Proto:      decoded.Proto,
		Headers:    decoded.Headers,
		Body:       rawMessageToBody(decoded.Body),
		Truncated:  decoded.Truncated,
	}
	return nil
}

// ResponseData captures the details of an HTTP response.
type ResponseData struct {
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body,omitempty"`
	Size       int64       `json:"size"`
	Truncated  bool        `json:"truncated,omitempty"`
}

// MarshalJSON implements json.Marshaler for ResponseData.
func (d ResponseData) MarshalJSON() ([]byte, error) {
	type responseData struct {
		StatusCode int             `json:"status_code"`
		Headers    http.Header     `json:"headers"`
		Body       json.RawMessage `json:"body,omitempty"`
		Size       int64           `json:"size"`
		Truncated  bool            `json:"truncated,omitempty"`
	}

	return json.Marshal(responseData{
		StatusCode: d.StatusCode,
		Headers:    d.Headers,
		Body:       bodyToRawMessage(d.Body),
		Size:       d.Size,
		Truncated:  d.Truncated,
	})
}

// UnmarshalJSON implements json.Unmarshaler for ResponseData.
func (d *ResponseData) UnmarshalJSON(data []byte) error {
	type responseData struct {
		StatusCode int             `json:"status_code"`
		Headers    http.Header     `json:"headers"`
		Body       json.RawMessage `json:"body,omitempty"`
		Size       int64           `json:"size"`
		Truncated  bool            `json:"truncated,omitempty"`
	}

	var decoded responseData
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*d = ResponseData{
		StatusCode: decoded.StatusCode,
		Headers:    decoded.Headers,
		Body:       rawMessageToBody(decoded.Body),
		Size:       decoded.Size,
		Truncated:  decoded.Truncated,
	}
	return nil
}

// bodyToRawMessage converts a byte slice to a json.RawMessage for serialization.
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
