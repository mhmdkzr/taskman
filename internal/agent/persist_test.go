package agent

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/zendev-sh/goai/provider"
)

func TestTranscriptToMessages(t *testing.T) {
	callID := "call_01"
	tr := &store.Transcript{
		Session: &store.Session{
			Model:        "deepseek-v4-flash",
			SystemPrompt: "You are helpful.",
		},
		Turns: []store.TurnTranscript{
			{
				Turn: store.Turn{Number: 1},
				Prompt: store.Message{Role: "user", Parts: []store.Part{
					{Type: "text", Text: "what time is it?"},
				}},
				Steps: []store.StepTranscript{
					{
						Step: store.Step{Number: 1},
						Messages: []store.Message{
							{Role: "assistant", Parts: []store.Part{
								{Type: "reasoning", Text: "I can run a command."},
								{Type: "tool-call", ToolCallID: callID, ToolName: "bash", ToolInput: `{"command":"date"}`},
							}},
							{Role: "tool", Parts: []store.Part{
								{Type: "tool-result", ToolCallID: callID, Text: `{"exit_code":0,"stdout":"now"}`},
							}},
						},
					},
				},
			},
			{
				Turn: store.Turn{Number: 2},
				Prompt: store.Message{Role: "user", Parts: []store.Part{
					{Type: "text", Text: "again please"},
				}},
			},
		},
	}

	msgs, err := transcriptToMessages(tr)
	if err != nil {
		t.Fatalf("transcriptToMessages: %v", err)
	}
	if len(msgs) != 5 {
		t.Fatalf("messages = %d, want 5 (system, user, assistant, tool, user)", len(msgs))
	}
	if msgs[0].Role != provider.RoleSystem || msgs[0].Content[0].Text != "You are helpful." {
		t.Errorf("system message = %+v", msgs[0])
	}
	if msgs[2].Role != provider.RoleAssistant || msgs[2].Content[1].Type != provider.PartToolCall {
		t.Errorf("assistant message = %+v", msgs[2])
	}
	tc := msgs[2].Content[1]
	if tc.ToolCallID != callID || tc.ToolName != "bash" || string(tc.ToolInput) != `{"command":"date"}` {
		t.Errorf("tool-call part = %+v", tc)
	}
	if msgs[3].Role != provider.RoleTool || msgs[3].Content[0].Type != provider.PartToolResult {
		t.Errorf("tool message = %+v", msgs[3])
	}
	if msgs[4].Role != provider.RoleUser || msgs[4].Content[0].Text != "again please" {
		t.Errorf("second user message = %+v", msgs[4])
	}
}

func TestMessageToStoreProviderOptions(t *testing.T) {
	opts := `{"reasoning_effort":"high"}`
	msg := store.Message{Role: "assistant", ProviderOptions: opts, Parts: []store.Part{
		{Type: "text", Text: "hi", ProviderOptions: `{"cache":true}`},
	}}

	pm, err := transcriptMessage(msg)
	if err != nil {
		t.Fatalf("transcriptMessage: %v", err)
	}
	if pm.ProviderOptions["reasoning_effort"] != "high" {
		t.Errorf("message provider options = %+v", pm.ProviderOptions)
	}
	if pm.Content[0].ProviderOptions["cache"] != true {
		t.Errorf("part provider options = %+v", pm.Content[0].ProviderOptions)
	}
}

func TestTranscriptMessageRemoteRef(t *testing.T) {
	msg := store.Message{Role: "assistant", Parts: []store.Part{{
		Type:      "file",
		RemoteRef: `{"provider":"openai","id":"file-1"}`,
	}}}
	pm, err := transcriptMessage(msg)
	if err != nil {
		t.Fatalf("transcriptMessage: %v", err)
	}
	if pm.Content[0].RemoteRef == nil || pm.Content[0].RemoteRef.ID != "file-1" || pm.Content[0].RemoteRef.Provider != "openai" {
		t.Errorf("remote ref = %+v", pm.Content[0].RemoteRef)
	}

	// "null" and empty mean no ref.
	msg2 := store.Message{Role: "assistant", Parts: []store.Part{{Type: "file", RemoteRef: "null"}}}
	pm2, err := transcriptMessage(msg2)
	if err != nil {
		t.Fatalf("transcriptMessage: %v", err)
	}
	if pm2.Content[0].RemoteRef != nil {
		t.Errorf("remote ref for 'null' = %+v", pm2.Content[0].RemoteRef)
	}

	// Invalid JSON is an error.
	bad := store.Message{Role: "assistant", Parts: []store.Part{{Type: "file", RemoteRef: "not json"}}}
	if _, err := transcriptMessage(bad); err == nil {
		t.Error("transcriptMessage with invalid remote ref: want error")
	}

	// Invalid provider options are an error.
	badOpts := store.Message{Role: "assistant", Parts: []store.Part{{Type: "text", ProviderOptions: "not json"}}}
	if _, err := transcriptMessage(badOpts); err == nil {
		t.Error("transcriptMessage with invalid provider options: want error")
	}
}

func TestMarshalJSONNil(t *testing.T) {
	var nilPtr *provider.RemoteFileRef
	var nilMap map[string]any
	var nilSlice []string
	if got := marshalJSON(nilPtr); got != "" {
		t.Errorf("marshalJSON(nil ptr) = %q, want empty", got)
	}
	if got := marshalJSON(nilMap); got != "" {
		t.Errorf("marshalJSON(nil map) = %q, want empty", got)
	}
	if got := marshalJSON(nilSlice); got != "" {
		t.Errorf("marshalJSON(nil slice) = %q, want empty", got)
	}
	if got := marshalJSON(&provider.RemoteFileRef{Provider: "p"}); got == "" {
		t.Error("marshalJSON(non-nil) = empty, want JSON")
	}
}

func TestMessagesToStoreOnlyNewMessages(t *testing.T) {
	// Simulates the new turn's slice that ContinueSession passes: the user
	// prompt plus the assistant/tool messages it produced (no history).
	steps := []Step{{Number: 1, FinishReason: "stop"}}
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: []provider.Part{{Type: provider.PartText, Text: "new prompt"}}},
		{Role: provider.RoleAssistant, Content: []provider.Part{{Type: provider.PartText, Text: "new answer"}}},
	}
	storeMessages := messagesToStore(steps, messages)
	if len(storeMessages) != 1 {
		t.Fatalf("messages = %d, want 1", len(storeMessages))
	}
	got := storeMessages[0].Parts[0].Text
	if got != "new answer" {
		t.Errorf("saved message = %q, want %q", got, "new answer")
	}
	if storeMessages[0].Number != 1 || storeMessages[0].FinishReason != "stop" {
		t.Errorf("message metadata = %+v", storeMessages[0])
	}
}
