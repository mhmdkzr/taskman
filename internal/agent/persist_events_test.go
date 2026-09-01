package agent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/usage"
	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "taskman.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	if err := store.Migrate(st.RW()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return st
}

// seedSession writes a session with the given number of turns via SaveRun /
// AppendRun and returns its id.
func seedSession(t *testing.T, st *store.Store, nTurns int) string {
	t.Helper()
	ctx := context.Background()
	id, err := store.NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := store.SaveRun(ctx, st.RW(), id, store.Run{Model: "fake-model", Prompt: "first prompt"}); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}
	run := store.Run{
		Model: "fake-model",
		Messages: []store.Message{{
			Role: "assistant", Number: 1, FinishReason: "stop",
			Parts: []store.Part{{Type: "text", Text: "first answer"}},
		}},
	}
	for i := 1; i < nTurns; i++ {
		run.Prompt = "prompt " + string(rune('a'+i))
		run.Messages[0].Parts[0].Text = "answer " + string(rune('a'+i))
		if err := store.AppendRun(ctx, st.RW(), id, run); err != nil {
			t.Fatalf("AppendRun: %v", err)
		}
	}
	return id
}

func persistOptions() Options {
	return Options{
		Config: config.AgentConfig{DBPath: "taskman-test.db"},
		Model:  "fake-model",
		Tools:  []goai.Tool{testClockTool()},
	}
}

// TestPersistRunWritesUnderResultSessionID verifies PersistRun stores the run
// under res.SessionID (the id generated at session start) and returns it.
func TestPersistRunWritesUnderResultSessionID(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	res := &Result{
		SessionID:    "sess-1",
		Text:         "hello",
		FinishReason: "stop",
		Steps:        []Step{{Number: 1, FinishReason: "stop", Usage: usage.TokenUsage{TotalTokens: 42}}},
		Messages: []provider.Message{{
			Role:    provider.RoleAssistant,
			Content: []provider.Part{{Type: provider.PartText, Text: "hello"}},
		}},
	}
	sessionID, err := PersistRun(ctx, st, persistOptions(), publisher.Publisher{}, "hi", res)
	if err != nil {
		t.Fatalf("PersistRun: %v", err)
	}
	if sessionID != "sess-1" {
		t.Errorf("session id = %q, want %q", sessionID, "sess-1")
	}

	tr, err := store.GetSessionTranscript(ctx, st.RO(), "sess-1")
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 1 || len(tr.Turns[0].Steps) != 1 {
		t.Fatalf("transcript = %+v", tr)
	}
	if tr.Session.Model != "fake-model" {
		t.Errorf("session model = %q", tr.Session.Model)
	}
	if tr.Turns[0].Steps[0].Step.Usage.TotalTokens != 42 {
		t.Errorf("step usage = %+v", tr.Turns[0].Steps[0].Step.Usage)
	}
}

// TestContinueSessionAppends verifies continuing a session persists the next
// turn under the existing session id.
func TestContinueSessionAppends(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	sessionID := seedSession(t, st, 1)

	opts := persistOptions()
	opts.model = &fakeModel{}

	res, id, err := ContinueSession(ctx, st, opts, publisher.Publisher{}, sessionID, "second prompt")
	if err != nil {
		t.Fatalf("ContinueSession: %v", err)
	}
	if id != sessionID {
		t.Errorf("session id = %q, want %q", id, sessionID)
	}
	if res.Text != "final answer" {
		t.Errorf("result text = %q", res.Text)
	}

	tr, err := store.GetSessionTranscript(ctx, st.RO(), sessionID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 2 {
		t.Fatalf("turns = %d, want 2", len(tr.Turns))
	}
	if tr.Turns[1].Prompt.Parts[0].Text != "second prompt" {
		t.Errorf("turn 2 prompt = %q", tr.Turns[1].Prompt.Parts[0].Text)
	}
}

// TestForkRunPersistsFork verifies forking creates a new session with the
// source's turns shared by reference and the fork's turn appended.
func TestForkRunPersistsFork(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	sourceID := seedSession(t, st, 2)

	opts := persistOptions()
	opts.model = &fakeModel{}

	_, newID, err := ForkRun(ctx, st, opts, publisher.Publisher{}, sourceID, 2, "fork prompt")
	if err != nil {
		t.Fatalf("ForkRun: %v", err)
	}
	if newID == "" {
		t.Fatal("empty fork session id")
	}

	src, err := store.GetSessionTranscript(ctx, st.RO(), sourceID)
	if err != nil {
		t.Fatalf("source transcript: %v", err)
	}
	fork, err := store.GetSessionTranscript(ctx, st.RO(), newID)
	if err != nil {
		t.Fatalf("fork transcript: %v", err)
	}
	if len(fork.Turns) != 2 {
		t.Fatalf("fork turns = %d, want 2", len(fork.Turns))
	}
	// By reference: the fork shares turn 1 (same turn id), no copy.
	if fork.Turns[0].Turn.ID != src.Turns[0].Turn.ID {
		t.Errorf("fork turn 1 does not share the source turn id; want a shared reference")
	}
	if fork.Turns[0].Prompt.Parts[0].Text != src.Turns[0].Prompt.Parts[0].Text {
		t.Errorf("fork turn 1 prompt = %q, want %q", fork.Turns[0].Prompt.Parts[0].Text, src.Turns[0].Prompt.Parts[0].Text)
	}
	if fork.Turns[1].Prompt.Parts[0].Text != "fork prompt" {
		t.Errorf("fork turn 2 prompt = %q", fork.Turns[1].Prompt.Parts[0].Text)
	}
}

// TestSubagentRunsPersistWithoutRunEvents verifies the sub-agent path appends
// directly to its own session (store.AppendRun) and never touches a parent
// run's session.
func TestSubagentRunsPersistWithoutRunEvents(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	res := &Result{SessionID: "parent", Text: "x", Steps: []Step{{Number: 1, FinishReason: "stop"}}}
	if _, err := PersistRun(ctx, st, persistOptions(), publisher.Publisher{}, "parent", res); err != nil {
		t.Fatalf("PersistRun: %v", err)
	}

	// Emulate the sub-agent path: store.CreateSession + AppendRun directly
	// (no persist.go wrapper), which is exactly what spawn does.
	subID, err := store.NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := store.CreateSession(ctx, st.RW(), subID, "fake-model", "medium", 10, "", nil); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	run := store.Run{
		Prompt: "sub",
		Messages: []store.Message{{
			Role: "assistant", Number: 1, FinishReason: "stop",
			Parts: []store.Part{{Type: "text", Text: "x"}},
		}},
	}
	if err := store.AppendRun(ctx, st.RW(), subID, run); err != nil {
		t.Fatalf("AppendRun: %v", err)
	}

	parent, err := store.GetSessionTranscript(ctx, st.RO(), "parent")
	if err != nil {
		t.Fatalf("parent transcript: %v", err)
	}
	if len(parent.Turns) != 1 {
		t.Fatalf("parent turns = %d, want 1 (sub-agent must not touch it)", len(parent.Turns))
	}
	sub, err := store.GetSessionTranscript(ctx, st.RO(), subID)
	if err != nil {
		t.Fatalf("sub transcript: %v", err)
	}
	if len(sub.Turns) != 1 {
		t.Fatalf("sub turns = %d, want 1", len(sub.Turns))
	}
}
