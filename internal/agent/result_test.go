package agent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/usage"
	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
)

// TestOptionsOutput verifies the serializable rendering of Options.
func TestOptionsOutput(t *testing.T) {
	o := Options{
		Model:           "some-model",
		ReasoningEffort: ReasoningEffortHigh,
		MaxSteps:        7,
		Tools:           []goai.Tool{testClockTool()},
	}
	out := optionsOutput(o)
	if out.Model != "some-model" || out.ReasoningEffort != ReasoningEffortHigh || out.MaxSteps != 7 {
		t.Errorf("options = %+v", out)
	}
	if len(out.Tools) != 1 || out.Tools[0].Name != "datetime" || len(out.Tools[0].InputSchema) == 0 {
		t.Errorf("tools = %+v", out.Tools)
	}
}

func TestNewRunOutput(t *testing.T) {
	o := Options{Model: "m", SystemPrompt: "sys", MaxSteps: 5}
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: []provider.Part{{Type: provider.PartText, Text: "sys"}}},
		{Role: provider.RoleUser, Content: []provider.Part{{Type: provider.PartText, Text: "prompt"}}},
		{Role: provider.RoleAssistant, Content: []provider.Part{
			{Type: provider.PartReasoning, Text: "think"},
			{Type: provider.PartToolCall, ToolCallID: "c1", ToolName: "bash", ToolInput: json.RawMessage(`{"command":"date"}`)},
		}},
		{Role: provider.RoleTool, Content: []provider.Part{{Type: provider.PartToolResult, ToolCallID: "c1", ToolOutput: "now"}}},
		{Role: provider.RoleAssistant, Content: []provider.Part{{Type: provider.PartText, Text: "answer"}}},
	}
	steps := []Step{
		{Number: 1, FinishReason: "tool-calls", ToolCalls: []ToolCall{{ID: "c1", Name: "bash", Input: `{"command":"date"}`}}, Usage: usage.TokenUsage{TotalTokens: 5}},
		{Number: 2, FinishReason: "stop", Usage: usage.TokenUsage{TotalTokens: 7}},
	}
	tokenUsage := usage.TokenUsage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	out := NewRunOutput("sess-1", o, tokenUsage, messages, steps)

	if out.SessionID != "sess-1" || out.ID == "" || out.Timestamp == "" {
		t.Errorf("run output header = %+v", out)
	}
	if out.Options.Model != "m" || out.Usage.TotalTokens != 15 {
		t.Errorf("options/usage = %+v / %+v", out.Options, out.Usage)
	}
	// Steps: pre-step (system+user), step 1 (assistant tool-call + tool), step 2.
	if len(out.Steps) != 3 {
		t.Fatalf("steps = %d, want 3", len(out.Steps))
	}
	last := out.Steps[2].Messages[0]
	if last.Number != 2 || last.FinishReason != "stop" || last.Usage == nil || last.Usage.TotalTokens != 7 {
		t.Errorf("last message = %+v", last)
	}
	toolMsg := out.Steps[1].Messages[1]
	if toolMsg.Content[0].ToolOutput != "now" {
		t.Errorf("tool message = %+v", toolMsg)
	}
}

func TestRemoteFileRefView(t *testing.T) {
	if v := remoteFileRefView(nil); v != nil {
		t.Errorf("nil ref -> %+v", v)
	}
	ref := &provider.RemoteFileRef{
		Provider:  "openai",
		ID:        "file-1",
		URI:       "https://example.com/f.png",
		Filename:  "f.png",
		MediaType: "image/png",
		ExpiresAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	v := remoteFileRefView(ref)
	if v.Provider != "openai" || v.ID != "file-1" || v.ExpiresAt != "2026-01-02T03:04:05Z" {
		t.Errorf("view = %+v", v)
	}
}
