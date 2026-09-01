package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
)

// PersistRun stores a completed run (session, turn, steps, messages and tool
// calls) as one atomic transaction, so a failure never leaves a partial run.
// res.SessionID (generated at session start) is the session id.
func PersistRun(ctx context.Context, st *store.Store, o Options, pub publisher.Publisher, prompt string, res *Result) (string, error) {
	run := store.Run{
		Model:           o.Model,
		ReasoningEffort: string(o.ReasoningEffort),
		MaxSteps:        o.MaxSteps,
		SystemPrompt:    o.SystemPrompt,
		Prompt:          prompt,
		Tools:           toolsToStore(o.Tools),
		Messages:        messagesToStore(res.Steps, res.Messages),
	}
	if err := store.SaveRun(ctx, st.RW(), res.SessionID, run); err != nil {
		return "", err
	}
	return res.SessionID, nil
}

// ContinueSession runs prompt in the context of an existing session's history
// and appends the outcome as the session's next turn. It returns the result
// and the (unchanged) session id.
func ContinueSession(ctx context.Context, st *store.Store, o Options, pub publisher.Publisher, sessionID, prompt string) (*Result, string, error) {
	tr, err := store.GetSessionTranscript(ctx, st.RW(), sessionID)
	if err != nil {
		return nil, "", err
	}

	messages, err := transcriptToMessages(tr)
	if err != nil {
		return nil, "", err
	}

	s, err := NewSession(o, pub, sessionID)
	if err != nil {
		return nil, "", err
	}
	s.messages = messages

	base := len(s.messages)
	res, err := s.Run(ctx, prompt)
	if err != nil {
		return nil, "", err
	}

	run := store.Run{
		Prompt:   prompt,
		Messages: messagesToStore(res.Steps, s.messages[base:]),
	}
	if err := store.AppendRun(ctx, st.RW(), sessionID, run); err != nil {
		return nil, "", err
	}
	return res, sessionID, nil
}

// ForkRun forks sourceID at atTurn: it creates a new session sharing the
// source's turns 1..atTurn-1 (referenced, not copied), runs newPrompt as the
// fork's turn atTurn, and persists the result. It returns the new session id
// and the result. Options are inherited from the source session; o supplies
// the provider config and registered tools.
func ForkRun(ctx context.Context, st *store.Store, o Options, pub publisher.Publisher, sourceID string, atTurn int, newPrompt string) (*Result, string, error) {
	tr, err := store.GetSessionTranscript(ctx, st.RW(), sourceID)
	if err != nil {
		return nil, "", err
	}
	if atTurn < 1 || atTurn > len(tr.Turns) {
		return nil, "", fmt.Errorf("turn %d out of range: session has %d turns", atTurn, len(tr.Turns))
	}

	newID, err := store.NewSessionID()
	if err != nil {
		return nil, "", fmt.Errorf("fork: generate session id: %w", err)
	}
	if err := store.ForkSession(ctx, st.RW(), sourceID, newID, atTurn); err != nil {
		return nil, "", err
	}

	forkOpts := o
	forkOpts.Model = tr.Session.Model
	forkOpts.ReasoningEffort = ReasoningEffort(tr.Session.ReasoningEffort)
	forkOpts.MaxSteps = tr.Session.MaxSteps
	forkOpts.SystemPrompt = tr.Session.SystemPrompt

	prefix := &store.Transcript{Session: tr.Session, Turns: tr.Turns[:atTurn-1]}
	messages, err := transcriptToMessages(prefix)
	if err != nil {
		return nil, "", err
	}

	s, err := NewSession(forkOpts, pub, newID)
	if err != nil {
		return nil, "", err
	}
	s.messages = messages

	base := len(s.messages)
	res, err := s.Run(ctx, newPrompt)
	if err != nil {
		return nil, "", err
	}

	run := store.Run{
		Prompt:   newPrompt,
		Messages: messagesToStore(res.Steps, s.messages[base:]),
	}
	if err := store.AppendRun(ctx, st.RW(), newID, run); err != nil {
		return nil, "", err
	}
	return res, newID, nil
}

// transcriptToMessages rebuilds the provider message history of a session:
// the system prompt, then each turn's user prompt and the messages its steps
// produced.
func transcriptToMessages(tr *store.Transcript) ([]provider.Message, error) {
	var out []provider.Message
	if tr.Session.SystemPrompt != "" {
		out = append(out, goai.SystemMessage(tr.Session.SystemPrompt))
	}
	for _, tt := range tr.Turns {
		msg, err := transcriptMessage(tt.Prompt)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
		for _, st := range tt.Steps {
			for _, m := range st.Messages {
				msg, err := transcriptMessage(m)
				if err != nil {
					return nil, err
				}
				out = append(out, msg)
			}
		}
	}
	return out, nil
}

func transcriptMessage(m store.Message) (provider.Message, error) {
	parts := make([]provider.Part, 0, len(m.Parts))
	for _, p := range m.Parts {
		pp := provider.Part{
			Type:            provider.PartType(p.Type),
			Text:            p.Text,
			URL:             p.URL,
			ToolCallID:      p.ToolCallID,
			ToolName:        p.ToolName,
			ToolInput:       json.RawMessage(p.ToolInput),
			ToolOutput:      p.ToolOutput,
			CacheControl:    p.CacheControl,
			CacheControlTTL: p.CacheControlTTL,
			Detail:          p.Detail,
			MediaType:       p.MediaType,
			Filename:        p.Filename,
		}
		if p.RemoteRef != "" && p.RemoteRef != "null" {
			var ref provider.RemoteFileRef
			if err := json.Unmarshal([]byte(p.RemoteRef), &ref); err != nil {
				return provider.Message{}, fmt.Errorf("unmarshal remote ref: %w", err)
			}
			pp.RemoteRef = &ref
		}
		if p.ProviderOptions != "" {
			if err := json.Unmarshal([]byte(p.ProviderOptions), &pp.ProviderOptions); err != nil {
				return provider.Message{}, fmt.Errorf("unmarshal provider options: %w", err)
			}
		}
		parts = append(parts, pp)
	}
	msg := provider.Message{Role: provider.Role(m.Role), Content: parts}
	if m.ProviderOptions != "" {
		if err := json.Unmarshal([]byte(m.ProviderOptions), &msg.ProviderOptions); err != nil {
			return provider.Message{}, fmt.Errorf("unmarshal message provider options: %w", err)
		}
	}
	return msg, nil
}

func toolsToStore(ts []goai.Tool) []store.Tool {
	out := make([]store.Tool, 0, len(ts))
	for _, t := range ts {
		out = append(out, store.Tool{
			Name:                t.Name,
			Description:         t.Description,
			InputSchema:         string(t.InputSchema),
			ProviderDefinedType: t.ProviderDefinedType,
		})
	}
	return out
}

// messagesToStore converts a transcript's assistant and tool messages into a
// flat store slice. Each assistant message opens a new step (aligned with the
// steps slice, which carries the per-step metadata); the tool messages that
// follow it carry that step's number. System and user messages are not included
// here: they are stored separately (the system message on the session, the user
// prompt as the turn's prompt).
func messagesToStore(steps []Step, messages []provider.Message) []store.Message {
	errByCallID := make(map[string]string)
	for _, st := range steps {
		for _, tc := range st.ToolCalls {
			if tc.Error != "" {
				errByCallID[tc.ID] = tc.Error
			}
		}
	}

	stepIdx := 0
	var out []store.Message
	for _, m := range messages {
		switch m.Role {
		case provider.RoleAssistant:
			if stepIdx >= len(steps) {
				continue
			}
			st := steps[stepIdx]
			stepIdx++
			out = append(out, store.Message{
				Role:            "assistant",
				Number:          st.Number,
				FinishReason:    st.FinishReason,
				Usage:           st.Usage,
				ProviderOptions: marshalJSON(m.ProviderOptions),
				Parts:           partsToStore(m.Content, errByCallID),
			})
		case provider.RoleTool:
			if stepIdx == 0 {
				continue
			}
			out = append(out, store.Message{
				Role:            "tool",
				Number:          steps[stepIdx-1].Number,
				ProviderOptions: marshalJSON(m.ProviderOptions),
				Parts:           partsToStore(m.Content, errByCallID),
			})
		}
	}
	return out
}

// partsToStore converts provider parts into store parts, attaching the tool
// error (from the step's tool calls) to tool-result parts.
func partsToStore(parts []provider.Part, errByCallID map[string]string) []store.Part {
	out := make([]store.Part, 0, len(parts))
	for _, p := range parts {
		sp := store.Part{
			Type:            string(p.Type),
			Text:            p.Text,
			URL:             p.URL,
			ToolCallID:      p.ToolCallID,
			ToolName:        p.ToolName,
			ToolInput:       string(p.ToolInput),
			ToolOutput:      p.ToolOutput,
			CacheControl:    p.CacheControl,
			CacheControlTTL: p.CacheControlTTL,
			Detail:          p.Detail,
			MediaType:       p.MediaType,
			Filename:        p.Filename,
			RemoteRef:       marshalJSON(p.RemoteRef),
			ProviderOptions: marshalJSON(p.ProviderOptions),
		}
		if p.Type == provider.PartToolResult {
			sp.ToolError = errByCallID[p.ToolCallID]
		}
		out = append(out, sp)
	}
	return out
}

// marshalJSON renders v as a JSON string, or "" when v is nil or marshals to
// nothing useful.
func marshalJSON(v any) string {
	if v == nil {
		return ""
	}
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface:
		if rv.IsNil() {
			return ""
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
