package agent

import (
	"context"
	"testing"

	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/zendev-sh/goai"
)

type testVerdict struct {
	Approved bool   `json:"approved" jsonschema:"description=Whether the change passes review."`
	Feedback string `json:"feedback,omitempty" jsonschema:"description=Why not, if rejected."`
}

// TestRunObjectParsesStructuredAnswer verifies RunObject calls tools like Run,
// then parses the final step's text into the requested type instead of
// returning it as free text.
func TestRunObjectParsesStructuredAnswer(t *testing.T) {
	ctx := context.Background()
	model := &fakeModel{
		toolName:  "datetime",
		toolArgs:  "{}",
		finalText: `{"approved": true, "feedback": ""}`,
	}
	opts := testOptions([]goai.Tool{testClockTool()}, model)

	verdict, res, err := RunObject[testVerdict](ctx, opts, publisher.Publisher{}, "review this", "")
	if err != nil {
		t.Fatalf("RunObject: %v", err)
	}
	if !verdict.Approved {
		t.Errorf("verdict = %+v, want Approved=true", verdict)
	}
	if res.SessionID == "" {
		t.Error("Result.SessionID is empty")
	}
	if len(res.Steps) != 2 {
		t.Errorf("Steps = %d, want 2 (tool call then final answer)", len(res.Steps))
	}
}

// TestContinueSessionObjectAppendsTurn verifies ContinueSessionObject loads an
// existing session's history, runs the structured turn, and appends it rather
// than starting fresh.
func TestContinueSessionObjectAppendsTurn(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	sessionID, err := store.NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := store.CreateSession(ctx, st.RW(), sessionID, "fake-model", "medium", 10, "you are a reviewer", nil); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	model := &fakeModel{finalText: `{"approved": false, "feedback": "needs a test"}`}
	opts := testOptions(nil, model)

	verdict, res, gotID, err := ContinueSessionObject[testVerdict](ctx, st, opts, publisher.Publisher{}, sessionID, "review this")
	if err != nil {
		t.Fatalf("ContinueSessionObject: %v", err)
	}
	if verdict.Approved || verdict.Feedback != "needs a test" {
		t.Errorf("verdict = %+v", verdict)
	}
	if gotID != sessionID {
		t.Errorf("session id = %q, want %q", gotID, sessionID)
	}
	if res.SessionID != sessionID {
		t.Errorf("Result.SessionID = %q, want %q", res.SessionID, sessionID)
	}

	tr, err := store.GetSessionTranscript(ctx, st.RW(), sessionID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 1 {
		t.Errorf("turns = %d, want 1", len(tr.Turns))
	}
}
