package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/mhmdkzr/app/pkg/middleware/auditlog"
)

const (
	// SubjectLogsDebug is the NATS subject for debug logs.
	SubjectLogsDebug = "system.logs.debug"
	// SubjectLogsInfo is the NATS subject for info logs.
	SubjectLogsInfo = "system.logs.info"
	// SubjectLogsWarn is the NATS subject for warn logs.
	SubjectLogsWarn = "system.logs.warn"
	// SubjectLogsError is the NATS subject for error logs.
	SubjectLogsError = "system.logs.error"
)

// NATSHandler wraps a base slog.Handler and publishes warn/error logs to NATS.
type NATSHandler struct {
	wrapped slog.Handler
	nc      *nats.Conn
	attrs   []slog.Attr
	groups  []string
}

// logMessage is the JSON structure published to NATS.
type logMessage struct {
	Timestamp  time.Time                  `json:"timestamp"`
	Level      string                     `json:"level"`
	Message    string                     `json:"message"`
	Attributes map[string]json.RawMessage `json:"attributes,omitempty"`
}

type logSource struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function,omitempty"`
}

// NewNATSHandler creates a new NATSHandler that wraps the given handler.
func NewNATSHandler(wrapped slog.Handler, nc *nats.Conn) *NATSHandler {
	return &NATSHandler{wrapped: wrapped, nc: nc}
}

// Enabled reports whether the handler handles records at the given level.
func (h *NATSHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.wrapped.Enabled(ctx, level)
}

// Handle handles the Record by passing it to the wrapped handler and,
// if the level is Warn or Error, publishing it to NATS.
func (h *NATSHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.wrapped.Handle(ctx, r); err != nil {
		return fmt.Errorf("handle wrapped slog record: %w", err)
	}

	if r.Level < slog.LevelWarn {
		return nil
	}

	msg := logMessage{
		Timestamp:  r.Time,
		Level:      r.Level.String(),
		Message:    r.Message,
		Attributes: h.buildAttributes(r),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		h.logError(ctx, "failed to marshal log message for NATS", err, "")
		return nil
	}

	subject := getSubject(r.Level)

	if err := h.nc.Publish(subject, data); err != nil {
		h.logError(ctx, "failed to publish log to NATS", err, subject)
	}

	return nil
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
func (h *NATSHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nextAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	nextAttrs = append(nextAttrs, h.attrs...)
	nextAttrs = append(nextAttrs, attrs...)

	return &NATSHandler{
		wrapped: h.wrapped.WithAttrs(attrs),
		nc:      h.nc,
		attrs:   nextAttrs,
		groups:  append([]string(nil), h.groups...),
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (h *NATSHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	nextGroups := make([]string, 0, len(h.groups)+1)
	nextGroups = append(nextGroups, h.groups...)
	nextGroups = append(nextGroups, name)

	return &NATSHandler{
		wrapped: h.wrapped.WithGroup(name),
		nc:      h.nc,
		attrs:   append([]slog.Attr(nil), h.attrs...),
		groups:  nextGroups,
	}
}

// getSubject returns the NATS subject for the given log level.
func getSubject(level slog.Level) string {
	switch {
	case level <= slog.LevelDebug:
		return SubjectLogsDebug
	case level < slog.LevelWarn:
		return SubjectLogsInfo
	case level < slog.LevelError:
		return SubjectLogsWarn
	default:
		return SubjectLogsError
	}
}

// buildAttributes builds the attribute map for a log record.
func (h *NATSHandler) buildAttributes(r slog.Record) map[string]json.RawMessage {
	attrs := make(map[string]json.RawMessage)

	for _, attr := range h.attrs {
		addAttr(attrs, joinGroups(h.groups, attr.Key), attr.Value)
	}

	r.Attrs(func(attr slog.Attr) bool {
		addAttr(attrs, joinGroups(h.groups, attr.Key), attr.Value)
		return true
	})
	addSource(attrs, r)

	if len(attrs) == 0 {
		return nil
	}

	return attrs
}

// addSource adds source file information to the attribute map.
func addSource(dst map[string]json.RawMessage, r slog.Record) {
	if r.PC == 0 {
		return
	}

	frames := runtime.CallersFrames([]uintptr{r.PC})
	frame, _ := frames.Next()
	if frame.File == "" {
		return
	}

	source := logSource{
		File: frame.File,
		Line: frame.Line,
	}
	if frame.Function != "" {
		source.Function = frame.Function
	}

	raw, err := json.Marshal(source)
	if err != nil {
		return
	}
	dst["source"] = raw
}

// addAttr adds a single attribute to the attribute map.
func addAttr(dst map[string]json.RawMessage, key string, value slog.Value) {
	if key == "" {
		return
	}

	if value.Kind() == slog.KindGroup {
		for _, nested := range value.Group() {
			addAttr(dst, joinGroups([]string{key}, nested.Key), nested.Value)
		}
		return
	}

	if auditlog.IsSensitiveKey(leafKey(key)) {
		raw, err := json.Marshal(auditlog.Marker)
		if err != nil {
			return
		}
		dst[key] = raw
		return
	}

	if errValue, ok := value.Any().(error); ok && errValue != nil {
		raw, err := json.Marshal(errValue.Error())
		if err != nil {
			return
		}
		dst[key] = raw
		return
	}

	raw, err := json.Marshal(value.Any())
	if err != nil {
		return
	}
	dst[key] = raw
}

// leafKey returns the last dot-separated segment of a joined attribute key,
// i.e. the attribute's own key with any group prefixes stripped. Sensitivity
// is checked against this leaf so a grouped key like "request.password" is
// still redacted based on "password".
func leafKey(key string) string {
	if idx := strings.LastIndex(key, "."); idx >= 0 {
		return key[idx+1:]
	}
	return key
}

// joinGroups joins group names with a key using dot notation.
func joinGroups(groups []string, key string) string {
	if len(groups) == 0 {
		return key
	}

	var result strings.Builder
	result.WriteString(groups[0])
	for _, group := range groups[1:] {
		result.WriteString("." + group)
	}
	return result.String() + "." + key
}

// logError logs an error from the NATS handler itself.
func (h *NATSHandler) logError(ctx context.Context, msg string, err error, subject string) {
	r := slog.NewRecord(time.Now(), slog.LevelError, msg, 0)
	r.AddAttrs(slog.String("error", err.Error()))
	if subject != "" {
		r.AddAttrs(slog.String("subject", subject))
	}

	if handleErr := h.wrapped.Handle(ctx, r); handleErr != nil {
		slog.Default().ErrorContext(ctx, "failed to log nats handler error", "error", handleErr)
	}
}
