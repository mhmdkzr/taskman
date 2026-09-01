package logger

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/mhmdkzr/taskman/pkg/natsembed"
	"github.com/mhmdkzr/taskman/pkg/testenv"
)

type decodedLogMessage struct {
	Timestamp  time.Time                  `json:"timestamp"`
	Level      string                     `json:"level"`
	Message    string                     `json:"message"`
	Attributes map[string]json.RawMessage `json:"attributes,omitempty"`
}

func TestNATSHandler_PublishesWarnAndError_Integration(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	t.Parallel()

	nc, _, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("connect embedded nats: %v", err)
	}
	t.Cleanup(nc.Close)

	warnSub, err := nc.SubscribeSync(SubjectLogsWarn)
	if err != nil {
		t.Fatalf("subscribe warn: %v", err)
	}
	errorSub, err := nc.SubscribeSync(SubjectLogsError)
	if err != nil {
		t.Fatalf("subscribe error: %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("flush subscriptions: %v", err)
	}

	h := NewNATSHandler(slog.DiscardHandler, nc).
		WithGroup("request").
		WithAttrs([]slog.Attr{slog.String("service", "app")})
	logger := slog.New(h)

	warnErr := errors.New("warn-path")
	logger.Warn("warn message", slog.Int("attempt", 2), slog.Any("err", warnErr))

	warnMsg, err := warnSub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("receive warn message: %v", err)
	}

	decodedWarn := decodeLogMessage(t, warnMsg)
	if decodedWarn.Level != "WARN" {
		t.Fatalf("warn level mismatch: got=%q want=%q", decodedWarn.Level, "WARN")
	}
	if decodedWarn.Message != "warn message" {
		t.Fatalf("warn message mismatch: got=%q", decodedWarn.Message)
	}
	assertAttrString(t, decodedWarn.Attributes, "request.service", "app")
	assertAttrInt(t, decodedWarn.Attributes, "request.attempt", 2)
	assertAttrString(t, decodedWarn.Attributes, "request.err", "warn-path")

	errErr := errors.New("error-path")
	logger.Error("error message", slog.Any("err", errErr), slog.String("kind", "fatal"))

	errMsg, err := errorSub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("receive error message: %v", err)
	}

	decodedErr := decodeLogMessage(t, errMsg)
	if decodedErr.Level != "ERROR" {
		t.Fatalf("error level mismatch: got=%q want=%q", decodedErr.Level, "ERROR")
	}
	if decodedErr.Message != "error message" {
		t.Fatalf("error message mismatch: got=%q", decodedErr.Message)
	}
	assertAttrString(t, decodedErr.Attributes, "request.service", "app")
	assertAttrString(t, decodedErr.Attributes, "request.kind", "fatal")
	assertAttrString(t, decodedErr.Attributes, "request.err", "error-path")
}

func TestNATSHandler_DoesNotPublishBelowWarn_Integration(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	t.Parallel()

	nc, _, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("connect embedded nats: %v", err)
	}
	t.Cleanup(nc.Close)

	debugSub, err := nc.SubscribeSync(SubjectLogsDebug)
	if err != nil {
		t.Fatalf("subscribe debug: %v", err)
	}
	infoSub, err := nc.SubscribeSync(SubjectLogsInfo)
	if err != nil {
		t.Fatalf("subscribe info: %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("flush subscriptions: %v", err)
	}

	h := NewNATSHandler(slog.DiscardHandler, nc)
	logger := slog.New(h)

	logger.Debug("debug message")
	logger.Info("info message")

	if _, err := debugSub.NextMsg(250 * time.Millisecond); !errors.Is(err, nats.ErrTimeout) {
		t.Fatalf("expected debug timeout, got: %v", err)
	}
	if _, err := infoSub.NextMsg(250 * time.Millisecond); !errors.Is(err, nats.ErrTimeout) {
		t.Fatalf("expected info timeout, got: %v", err)
	}
}

// TestNATSHandler_RedactsSensitiveAttributes_Integration pins the redaction
// contract added for bugs.md finding #13: NATSHandler applies sensitive-key
// redaction (IsSensitiveKey) to every Warn/Error attribute before publishing.
// This test logs an Error whose attributes include sensitive keys (e.g.
// "password", "api_key") with real-looking secret values, and checks what
// actually reaches a subscriber on the NATS error subject.
//
// Both probe keys are redacted to the Marker before publishing, so the raw
// secret values never reach the wire and the test passes.
func TestNATSHandler_RedactsSensitiveAttributes_Integration(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	t.Parallel()

	nc, _, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("connect embedded nats: %v", err)
	}
	t.Cleanup(nc.Close)

	errorSub, err := nc.SubscribeSync(SubjectLogsError)
	if err != nil {
		t.Fatalf("subscribe error: %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("flush subscriptions: %v", err)
	}

	// Deliberately not slog.DiscardHandler here: its Enabled() is documented
	// to return false for every level, which makes slog.Logger.Warn/Error
	// skip calling NATSHandler.Handle entirely (Logger checks Enabled before
	// invoking Handle) — so publishing never happens and this test would
	// hang on NextMsg for reasons unrelated to the bug under test.
	h := NewNATSHandler(alwaysEnabledDiscardHandler{}, nc)
	logger := slog.New(h)

	const secretPassword = "s3cr3t-db-password-do-not-leak"
	const secretAPIKey = "sk_live_do-not-leak-this-either"
	logger.Error("failed to connect",
		slog.String("password", secretPassword),
		slog.String("api_key", secretAPIKey),
	)

	msg, err := errorSub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("receive error message: %v", err)
	}
	decoded := decodeLogMessage(t, msg)

	for key, leaked := range map[string]string{
		"password": secretPassword,
		"api_key":  secretAPIKey,
	} {
		raw, ok := decoded.Attributes[key]
		if !ok {
			continue
		}
		var got string
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("decode attr %q: %v", key, err)
		}
		if got == leaked {
			t.Fatalf(
				"NATSHandler published the raw secret value for attribute %q "+
					"(%q) to the NATS error subject, unredacted — Warn/Error attributes "+
					"forwarded to Telegram must be redacted (bugs.md finding #13)",
				key, got,
			)
		}
	}
}

func TestInit_WithNATSEnabled_PublishesServiceAttr_Integration(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	nc, _, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("connect embedded nats: %v", err)
	}
	t.Cleanup(nc.Close)

	sub, err := nc.SubscribeSync(SubjectLogsWarn)
	if err != nil {
		t.Fatalf("subscribe warn: %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("flush subscriptions: %v", err)
	}

	prev := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(prev)
	})

	cfg := Config{Format: LogFormatJSON, Level: slog.LevelDebug}
	if err := Init(cfg, nc); err != nil {
		t.Fatalf("init logger: %v", err)
	}

	slog.WarnContext(context.Background(), "integration warning", slog.String("scope", "init"))

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("receive warn message: %v", err)
	}
	decoded := decodeLogMessage(t, msg)
	assertAttrString(t, decoded.Attributes, "service", "app-service")
	assertAttrString(t, decoded.Attributes, "scope", "init")
}

// alwaysEnabledDiscardHandler is a no-op slog.Handler whose Enabled always
// returns true, unlike slog.DiscardHandler (see the comment where this is
// used). Only Handle needs to be a true no-op for these tests; NATSHandler
// never calls WithAttrs/WithGroup on tests that don't chain them.
type alwaysEnabledDiscardHandler struct{}

func (alwaysEnabledDiscardHandler) Enabled(context.Context, slog.Level) bool  { return true }
func (alwaysEnabledDiscardHandler) Handle(context.Context, slog.Record) error { return nil }
func (h alwaysEnabledDiscardHandler) WithAttrs([]slog.Attr) slog.Handler      { return h }
func (h alwaysEnabledDiscardHandler) WithGroup(string) slog.Handler           { return h }

// TestNATSHandler_LogErrorDoesNotRecurse_Integration covers the logError level
// change (Info -> Error): logError must call the wrapped handler directly
// rather than going back through NATSHandler.Handle, otherwise logging its
// own publish failure at Error level would attempt to publish again and
// recurse forever. It forces a publish failure by closing the NATS
// connection before logging, then asserts the wrapped handler saw exactly
// one extra record (the self-log) and nothing beyond it.
func TestNATSHandler_LogErrorDoesNotRecurse_Integration(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	t.Parallel()

	nc, _, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("connect embedded nats: %v", err)
	}
	nc.Close()

	wrapped := &recordingHandler{}
	h := NewNATSHandler(wrapped, nc)
	logger := slog.New(h)

	done := make(chan struct{})
	go func() {
		logger.Error("boom", slog.String("k", "v"))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Handle did not return: possible infinite recursion in logError")
	}

	wrapped.mu.Lock()
	defer wrapped.mu.Unlock()
	if len(wrapped.records) != 2 {
		t.Fatalf("expected exactly 2 records (original + one self-log), got %d", len(wrapped.records))
	}
	if wrapped.records[0].Message != "boom" {
		t.Fatalf("unexpected first record message: %q", wrapped.records[0].Message)
	}
	selfLog := wrapped.records[1]
	if selfLog.Level != slog.LevelError {
		t.Fatalf("expected self-log level ERROR, got %s", selfLog.Level)
	}
	if selfLog.Message != "failed to publish log to NATS" {
		t.Fatalf("unexpected self-log message: %q", selfLog.Message)
	}
}

// recordingHandler is a slog.Handler that stores every record it receives,
// used to detect recursive re-entry into NATSHandler.Handle.
type recordingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

func decodeLogMessage(t *testing.T, msg *nats.Msg) decodedLogMessage {
	t.Helper()

	var out decodedLogMessage
	if err := json.Unmarshal(msg.Data, &out); err != nil {
		t.Fatalf("decode log message: %v", err)
	}
	return out
}

func assertAttrString(t *testing.T, attrs map[string]json.RawMessage, key, want string) {
	t.Helper()

	raw, ok := attrs[key]
	if !ok {
		t.Fatalf("missing attr %q", key)
	}

	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode attr %q: %v", key, err)
	}
	if got != want {
		t.Fatalf("attr %q mismatch: got=%q want=%q", key, got, want)
	}
}

func assertAttrInt(t *testing.T, attrs map[string]json.RawMessage, key string, want int) {
	t.Helper()

	raw, ok := attrs[key]
	if !ok {
		t.Fatalf("missing attr %q", key)
	}

	var got int
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode attr %q: %v", key, err)
	}
	if got != want {
		t.Fatalf("attr %q mismatch: got=%d want=%d", key, got, want)
	}
}
