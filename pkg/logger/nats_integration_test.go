package logger

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/mhmdkzr/app/pkg/embeddednats"
	"github.com/mhmdkzr/app/pkg/testenv"
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

	nc, _, err := embeddednats.Connect()
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
		WithAttrs([]slog.Attr{slog.String("service", "core")})
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
	assertAttrString(t, decodedWarn.Attributes, "request.service", "core")
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
	assertAttrString(t, decodedErr.Attributes, "request.service", "core")
	assertAttrString(t, decodedErr.Attributes, "request.kind", "fatal")
	assertAttrString(t, decodedErr.Attributes, "request.err", "error-path")
}

func TestNATSHandler_DoesNotPublishBelowWarn_Integration(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	t.Parallel()

	nc, _, err := embeddednats.Connect()
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

func TestInit_WithNATSEnabled_PublishesServiceAttr_Integration(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	nc, _, err := embeddednats.Connect()
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
	assertAttrString(t, decoded.Attributes, "service", "core-service")
	assertAttrString(t, decoded.Attributes, "scope", "init")
}

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
