package logger

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/mhmdkzr/app/pkg/embeddednats"
)

func TestInit_ReturnsFormatError(t *testing.T) {
	err := Init(Config{Format: LogFormat("bad")}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errInvalidLogFormat) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInit_WithNilNATS(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	if err := Init(Config{Format: LogFormatText, Level: slog.LevelInfo}, nil); err != nil {
		t.Fatalf("init: %v", err)
	}
}

func TestGetSubject(t *testing.T) {
	cases := []struct {
		level slog.Level
		want  string
	}{
		{level: slog.LevelDebug, want: SubjectLogsDebug},
		{level: slog.LevelInfo, want: SubjectLogsInfo},
		{level: slog.LevelWarn, want: SubjectLogsWarn},
		{level: slog.LevelError, want: SubjectLogsError},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			got, err := getSubject(tc.level)
			if err != nil {
				t.Fatalf("getSubject: %v", err)
			}
			if got != tc.want {
				t.Fatalf("subject mismatch: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestNATSHandler_Handle(t *testing.T) {
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
		t.Fatalf("flush: %v", err)
	}

	h := NewNATSHandler(slog.DiscardHandler, nc).
		WithGroup("request").
		WithAttrs([]slog.Attr{slog.String("service", "core")})

	if err := h.Handle(
		context.Background(),
		slog.NewRecord(time.Now(), slog.LevelWarn, "warn message", 0),
	); err != nil {
		t.Fatalf("handle: %v", err)
	}
	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("next msg: %v", err)
	}
	if !strings.Contains(string(msg.Data), "warn message") {
		t.Fatalf("unexpected log payload: %s", string(msg.Data))
	}
}

func TestTemporalLogger_PromotesTaskFailureInfoToError(t *testing.T) {
	var out strings.Builder
	base := slog.New(slog.NewTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logger := NewTemporalLogger(base)
	logger.Info("Task processing failed with error", "Error", errors.New("boom"))

	got := out.String()
	if !strings.Contains(got, "level=ERROR") {
		t.Fatalf("expected error level, got log output: %s", got)
	}
	if !strings.Contains(got, "Task processing failed with error") {
		t.Fatalf("expected task failure message, got log output: %s", got)
	}
}
