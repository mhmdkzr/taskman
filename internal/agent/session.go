package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/deepseek"
)

// Session runs a multi-turn conversation, keeping message history across
// calls to Run and Stream. A Session is not safe for concurrent use.
type Session struct {
	opts      Options
	pub       publisher.Publisher
	messages  []provider.Message
	sessionID string
}

// publishTimeout bounds each event publish so a stalled bus cannot block the
// agent loop.
const publishTimeout = 5 * time.Second

// NewSession creates a session for the given options. Events are published to
// the bus through pub. Tools are used as given; build the standard set with
// DefaultTools.
//
// sessionID identifies the session across the event bus and the store. Pass an
// existing session's id to continue or fork it; pass "" to start a fresh
// session, whose id is generated here (a UUIDv7 whose embedded timestamp
// records session start). The id is available on the session's Result.
func NewSession(opts Options, pub publisher.Publisher, sessionID string) (*Session, error) {
	opts, err := normalize(opts)
	if err != nil {
		return nil, err
	}
	if sessionID == "" {
		sessionID, err = store.NewSessionID()
		if err != nil {
			return nil, fmt.Errorf("new session: %w", err)
		}
	}
	s := &Session{opts: opts, pub: pub, sessionID: sessionID}
	if opts.SystemPrompt != "" {
		s.messages = append(s.messages, goai.SystemMessage(opts.SystemPrompt))
	}
	return s, nil
}

// Run executes a single turn non-streaming and returns the final result.
func (s *Session) Run(ctx context.Context, prompt string) (*Result, error) {
	s.messages = append(s.messages, goai.UserMessage(prompt))
	s.publish(events.TurnStarted{SessionID: s.sessionID, Prompt: prompt})
	r, err := goai.GenerateText(ctx, s.model(), s.goaiOptions(s.messages)...)
	if err != nil {
		return nil, err
	}
	if r != nil && len(r.ResponseMessages) > 0 {
		s.messages = append(s.messages, r.ResponseMessages...)
	}
	res := fromGoaiResult(r)
	res.Messages = s.messages
	res.SessionID = s.sessionID
	s.publish(events.TurnFinished{
		SessionID:    s.sessionID,
		Prompt:       prompt,
		Text:         res.Text,
		Reasoning:    res.Reasoning,
		FinishReason: res.FinishReason,
		Usage:        usageToEvent(res.Usage),
		Steps:        len(res.Steps),
	})
	return res, nil
}

// Stream starts a single turn and returns an event stream. Consume the
// stream's Events channel to completion, then check Err and, on success, call
// Result and pass it to Append to record the turn.
func (s *Session) Stream(ctx context.Context, prompt string) (*EventStream, error) {
	s.messages = append(s.messages, goai.UserMessage(prompt))
	s.publish(events.TurnStarted{SessionID: s.sessionID, Prompt: prompt})
	results := newToolResults()
	opts := append(s.goaiOptions(s.messages), goai.WithOnToolCall(results.store))
	ts, err := goai.StreamText(ctx, s.model(), opts...)
	if err != nil {
		return nil, err
	}
	return newEventStream(ts, results, s.sessionID), nil
}

// Append records a finished turn (obtained from Stream.Result) in the session
// history. Only call with the result of a successfully drained stream.
func (s *Session) Append(result *Result) {
	if result != nil && len(result.Messages) > 0 {
		s.messages = append(s.messages, result.Messages...)
	}
}

// model returns the provider model for the configured model ID.
func (s *Session) model() provider.LanguageModel {
	if s.opts.model != nil {
		return s.opts.model
	}
	var opts []deepseek.Option // NOTE: hardcoded to use deepseek. might want to make it generic
	if s.opts.Config.Provider.APIKey != "" {
		opts = append(opts, deepseek.WithAPIKey(s.opts.Config.Provider.APIKey))
	}
	if s.opts.Config.Provider.BaseURL != "" {
		opts = append(opts, deepseek.WithBaseURL(s.opts.Config.Provider.BaseURL))
	}
	return deepseek.Chat(s.opts.Model, opts...)
}

// goaiOptions maps Options onto goai generation options. It registers the
// goai hooks that translate model-call and tool-loop callbacks into events
// published on the configured event bus.
func (s *Session) goaiOptions(messages []provider.Message) []goai.Option {
	return []goai.Option{
		goai.WithMessages(messages...),
		goai.WithTools(s.opts.Tools...),
		goai.WithProviderOptions(map[string]any{"reasoning_effort": string(s.opts.ReasoningEffort)}),
		goai.WithMaxSteps(s.opts.MaxSteps),
		goai.WithOnRequest(func(r goai.RequestInfo) {
			s.publish(events.Request{
				SessionID:    s.sessionID,
				Model:        r.Model,
				MessageCount: r.MessageCount,
				ToolCount:    r.ToolCount,
				Timestamp:    r.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			})
		}),
		goai.WithOnResponse(func(r goai.ResponseInfo) {
			ev := events.Response{
				SessionID:    s.sessionID,
				Latency:      r.Latency.String(),
				Usage:        usageEvent(r.Usage),
				FinishReason: string(r.FinishReason),
				StatusCode:   r.StatusCode,
			}
			if r.Error != nil {
				ev.Error = r.Error.Error()
			}
			s.publish(ev)
		}),
		goai.WithOnStepFinish(func(sr goai.StepResult) {
			calls := make([]events.ToolCall, 0, len(sr.ToolCalls))
			for _, tc := range sr.ToolCalls {
				calls = append(calls, toolCallEvent(tc))
			}
			s.publish(events.StepFinished{
				SessionID:    s.sessionID,
				Number:       sr.Number,
				Text:         sr.Text,
				Reasoning:    sr.Reasoning,
				ToolCalls:    calls,
				FinishReason: string(sr.FinishReason),
				Usage:        usageEvent(sr.Usage),
			})
		}),
		goai.WithOnToolCallStart(func(t goai.ToolCallStartInfo) {
			s.publish(events.ToolCallStarted{
				SessionID: s.sessionID,
				ID:        t.ToolCallID,
				Name:      t.ToolName,
				Step:      t.Step,
				Input:     string(t.Input),
			})
		}),
		goai.WithOnToolCall(func(t goai.ToolCallInfo) {
			ev := events.ToolCalled{
				SessionID: s.sessionID,
				ID:        t.ToolCallID,
				Name:      t.ToolName,
				Step:      t.Step,
				Input:     string(t.Input),
				Output:    t.Output,
				Duration:  t.Duration.String(),
				Skipped:   t.Skipped,
			}
			if t.Error != nil {
				ev.Error = t.Error.Error()
			}
			s.publish(ev)
		}),
		goai.WithOnFinish(func(f goai.FinishInfo) {
			s.publish(events.GenerationFinished{
				SessionID:      s.sessionID,
				StepsExhausted: f.StepsExhausted,
				TotalSteps:     f.TotalSteps,
				TotalUsage:     usageEvent(f.TotalUsage),
				FinishReason:   string(f.FinishReason),
				StoppedBy:      string(f.StoppedBy),
			})
		}),
		goai.WithOnPanic(func(pi goai.PanicInfo) {
			s.publish(events.Panic{
				SessionID: s.sessionID,
				Phase:     pi.Phase,
				Value:     fmt.Sprint(pi.Value),
				Stack:     string(pi.Stack),
			})
		}),
	}
}

// publish sends an event to the configured event bus. Publish errors are
// best-effort and ignored.
func (s *Session) publish(e publisher.Event[any]) {
	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()
	_ = s.pub.Publish(ctx, e)
}
