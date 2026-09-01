package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
)

// fakeModel simulates a model that first requests one tool call, then
// answers. finalText overrides the final step's text (e.g. to a JSON blob for
// GenerateObject); empty uses "final answer".
type fakeModel struct {
	toolName  string
	toolArgs  string
	finalText string
	calls     int
}

func (f *fakeModel) ModelID() string { return "fake-model" }

func (f *fakeModel) DoGenerate(ctx context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	f.calls++
	if f.calls == 1 && f.toolName != "" {
		return &provider.GenerateResult{
			FinishReason: provider.FinishToolCalls,
			ToolCalls: []provider.ToolCall{
				{ID: "call-1", Name: f.toolName, Input: json.RawMessage(f.toolArgs)},
			},
			Usage: provider.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15, ReasoningTokens: 2},
		}, nil
	}
	text := f.finalText
	if text == "" {
		text = "final answer"
	}
	return &provider.GenerateResult{
		Text:         text,
		FinishReason: provider.FinishStop,
		Usage:        provider.Usage{InputTokens: 20, OutputTokens: 5, TotalTokens: 25},
	}, nil
}

func (f *fakeModel) DoStream(ctx context.Context, params provider.GenerateParams) (*provider.StreamResult, error) {
	return nil, nil
}

func testOptions(tools []goai.Tool, model provider.LanguageModel) Options {
	return Options{
		Config: config.AgentConfig{DBPath: "taskman-test.db"},
		Model:  "fake-model",
		Tools:  tools,
		model:  model,
	}
}

// TestSessionRunEvents verifies a non-streaming turn publishes the full event
// sequence in order: turn start, per-step model call and step events, tool
// events around execution, generation finish, then turn finish.
func TestSessionRunEvents(t *testing.T) {
	ctx := context.Background()

	rec, pub := startRecorder(t)
	s, err := NewSession(testOptions([]goai.Tool{testClockTool()}, &fakeModel{toolName: "datetime", toolArgs: "{}"}), pub, "")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	res, err := s.Run(ctx, "what time is it?")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.SessionID == "" {
		t.Fatal("empty session id on result")
	}
	// The terminal event is the barrier: once it has been delivered, every
	// earlier publish has been delivered too (per-subscription ordering).
	rec.waitFor(t, "agent.turn.finished", 5*time.Second)

	want := []string{
		"agent.turn.started",
		"agent.model.request",
		"agent.model.response",
		"agent.step.finished",
		"agent.tool.started",
		"agent.tool.finished",
		"agent.model.request",
		"agent.model.response",
		"agent.step.finished",
		"agent.generation.finished",
		"agent.turn.finished",
	}
	got := rec.subjectsSnapshot()
	if len(got) != len(want) {
		t.Fatalf("subjects = %v\nwant %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("subject[%d] = %q, want %q\nall: %v", i, got[i], want[i], got)
		}
	}

	var started events.TurnStarted
	rec.decode(t, "agent.turn.started", &started)
	if started.Prompt != "what time is it?" {
		t.Errorf("TurnStarted.Prompt = %q", started.Prompt)
	}
	if started.SessionID != res.SessionID {
		t.Errorf("TurnStarted.SessionID = %q, want %q", started.SessionID, res.SessionID)
	}

	var finished events.TurnFinished
	rec.decode(t, "agent.turn.finished", &finished)
	if finished.Text != "final answer" || finished.FinishReason != "stop" {
		t.Errorf("TurnFinished = %+v", finished)
	}
	if finished.SessionID != res.SessionID {
		t.Errorf("TurnFinished.SessionID = %q, want %q", finished.SessionID, res.SessionID)
	}
	if finished.Steps != 2 {
		t.Errorf("TurnFinished.Steps = %d, want 2", finished.Steps)
	}
	if finished.Usage.TotalTokens != 40 {
		t.Errorf("TurnFinished.Usage = %+v, want total 40", finished.Usage)
	}
	if res.Text != "final answer" {
		t.Errorf("result text = %q", res.Text)
	}

	var tool events.ToolCalled
	rec.decode(t, "agent.tool.finished", &tool)
	if tool.Name != "datetime" || tool.Step != 1 || tool.Error != "" {
		t.Errorf("ToolCalled = %+v", tool)
	}
	if tool.SessionID != res.SessionID {
		t.Errorf("ToolCalled.SessionID = %q, want %q", tool.SessionID, res.SessionID)
	}

	var request events.Request
	rec.decode(t, "agent.model.request", &request)
	if request.SessionID != res.SessionID {
		t.Errorf("Request.SessionID = %q, want %q", request.SessionID, res.SessionID)
	}
}

// TestSessionRunPanicEvent verifies a panicking tool is surfaced through the
// OnPanic hook as an agent.panic event.
func TestSessionRunPanicEvent(t *testing.T) {
	ctx := context.Background()

	boom := goai.NewTool("boom", "always panics", func(ctx context.Context, in struct{}) (string, error) {
		panic("kaboom")
	})

	rec, pub := startRecorder(t)
	s, err := NewSession(testOptions([]goai.Tool{boom}, &fakeModel{toolName: "boom", toolArgs: "{}"}), pub, "")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if _, err := s.Run(ctx, "make it explode"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	rec.waitFor(t, "agent.panic", 5*time.Second)
	var p events.Panic
	rec.decode(t, "agent.panic", &p)
	if p.Phase == "" || p.Value != "kaboom" {
		t.Errorf("Panic = %+v", p)
	}
	if p.SessionID == "" {
		t.Errorf("Panic.SessionID is empty")
	}
}

// TestSessionAppend records a finished turn's messages in session history.
func TestSessionAppend(t *testing.T) {
	s := &Session{}
	s.Append(nil)
	if len(s.messages) != 0 {
		t.Fatalf("messages after nil append = %d", len(s.messages))
	}
	res := &Result{Messages: []provider.Message{
		{Role: provider.RoleAssistant, Content: []provider.Part{{Type: provider.PartText, Text: "hi"}}},
	}}
	s.Append(res)
	if len(s.messages) != 1 {
		t.Fatalf("messages after append = %d", len(s.messages))
	}
	s.Append(&Result{})
	if len(s.messages) != 1 {
		t.Fatalf("messages after empty append = %d", len(s.messages))
	}
}

// streamModel emits a fixed text stream with no tool calls.
type streamModel struct {
	text  string
	calls int
}

func (m *streamModel) ModelID() string { return "fake-stream" }

func (m *streamModel) DoGenerate(ctx context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	return nil, nil
}

func (m *streamModel) DoStream(ctx context.Context, params provider.GenerateParams) (*provider.StreamResult, error) {
	m.calls++
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		if m.calls == 1 {
			// First step: request the datetime tool, then stop the step.
			ch <- provider.StreamChunk{Type: provider.ChunkToolCall, ToolCallID: "call-s1", ToolName: "datetime", ToolInput: "{}"}
			ch <- provider.StreamChunk{Type: provider.ChunkFinish, FinishReason: provider.FinishToolCalls}
			return
		}
		// Second step: the final answer.
		ch <- provider.StreamChunk{Type: provider.ChunkText, Text: m.text}
		ch <- provider.StreamChunk{Type: provider.ChunkFinish, FinishReason: provider.FinishStop, Usage: provider.Usage{TotalTokens: 30}}
	}()
	return &provider.StreamResult{Stream: ch}, nil
}

// TestSessionStreamPublishesTurnStarted verifies the streaming path publishes
// turn.started and still yields a usable event stream.
func TestSessionStreamPublishesTurnStarted(t *testing.T) {
	ctx := context.Background()

	rec, pub := startRecorder(t)
	s, err := NewSession(testOptions([]goai.Tool{testClockTool()}, &streamModel{text: "streamed answer"}), pub, "")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	stream, err := s.Stream(ctx, "stream this")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var text string
	var toolCalls []ToolCall
	for ev := range stream.Events() {
		switch ev.Type {
		case EventText:
			text += ev.Text
		case EventToolCall:
			toolCalls = append(toolCalls, ev.ToolCall)
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream.Err: %v", err)
	}
	if text != "streamed answer" {
		t.Errorf("streamed text = %q", text)
	}
	if len(toolCalls) != 1 || toolCalls[0].Name != "datetime" || toolCalls[0].Error != "" {
		t.Errorf("tool calls = %+v", toolCalls)
	}
	if res := stream.Result(); res == nil || res.Text != "streamed answer" {
		t.Errorf("stream result = %+v", res)
	}
	if res := stream.Result(); res == nil || res.SessionID == "" {
		t.Errorf("stream result missing session id: %+v", res)
	}

	rec.waitFor(t, "agent.turn.started", 5*time.Second)
	var started events.TurnStarted
	rec.decode(t, "agent.turn.started", &started)
	if started.Prompt != "stream this" {
		t.Errorf("TurnStarted.Prompt = %q", started.Prompt)
	}
	if started.SessionID == "" {
		t.Errorf("TurnStarted.SessionID is empty")
	}
	rec.waitFor(t, "agent.tool.finished", 5*time.Second)
	var tool events.ToolCalled
	rec.decode(t, "agent.tool.finished", &tool)
	if tool.Name != "datetime" || tool.Error != "" {
		t.Errorf("ToolCalled = %+v", tool)
	}
}
