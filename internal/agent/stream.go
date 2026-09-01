package agent

import (
	"sync"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
)

// EventType identifies the kind of an Event.
type EventType int

const (
	// EventText is a chunk of the answer.
	EventText EventType = iota

	// EventReasoning is a chunk of the model's thinking.
	EventReasoning

	// EventToolCall carries a completed tool invocation and its outcome.
	EventToolCall

	// EventError reports a stream error.
	EventError

	// EventFinish marks the end of the stream.
	EventFinish
)

// Event is a single unit of a streaming turn.
type Event struct {
	// Type is the kind of event.
	Type EventType

	// Text holds content for EventText and EventReasoning.
	Text string

	// ToolCall holds the invocation and outcome for EventToolCall.
	ToolCall ToolCall

	// Error holds the error for EventError.
	Error error
}

// EventStream delivers the events of a single turn. Events must be consumed
// to completion before calling Err or Result.
type EventStream struct {
	ts        *goai.TextStream
	events    <-chan Event
	results   *toolResults
	sessionID string
}

func newEventStream(ts *goai.TextStream, results *toolResults, sessionID string) *EventStream {
	s := &EventStream{ts: ts, results: results, sessionID: sessionID}
	s.events = s.pump()
	return s
}

// Events returns the channel of events for this turn. It is closed when the
// turn ends. Drain it fully before calling Err or Result.
func (s *EventStream) Events() <-chan Event { return s.events }

// Err returns the first stream error, or nil. Must be called after Events has
// been fully drained.
func (s *EventStream) Err() error { return s.ts.Err() }

// Result returns the accumulated result for this turn. Must be called after
// Events has been fully drained.
func (s *EventStream) Result() *Result {
	res := fromGoaiResult(s.ts.Result())
	if res != nil && s.sessionID != "" {
		res.SessionID = s.sessionID
	}
	return res
}

// pump consumes the underlying goai stream and forwards agent-owned events.
// It is the sole writer to the events channel. Tool events are emitted only
// once the tool's outcome is known (from the OnToolCall hook), which keeps
// them ordered with the surrounding text.
func (s *EventStream) pump() <-chan Event {
	events := make(chan Event, 64)

	// Wake any wait blocked on a tool result that never executes (e.g. a
	// provider error after emitting a tool call). ts.Err() returns once the
	// stream is fully consumed.
	go func() {
		s.ts.Err()
		s.results.close()
	}()

	go func() {
		defer close(events)
		defer s.results.close()
		for chunk := range s.ts.Stream() {
			switch chunk.Type {
			case provider.ChunkText:
				events <- Event{Type: EventText, Text: chunk.Text}

			case provider.ChunkReasoning:
				events <- Event{Type: EventReasoning, Text: chunk.Text}

			case provider.ChunkToolCall:
				tc := ToolCall{
					ID:    chunk.ToolCallID,
					Name:  chunk.ToolName,
					Input: chunk.ToolInput,
				}
				if info, ok := s.results.wait(chunk.ToolCallID); ok {
					tc.Output = info.Output
					if info.Error != nil {
						tc.Error = info.Error.Error()
					}
				} else {
					tc.Error = "tool call not executed"
				}
				events <- Event{Type: EventToolCall, ToolCall: tc}

			case provider.ChunkError:
				if chunk.Error != nil {
					events <- Event{Type: EventError, Error: chunk.Error}
				}

			case provider.ChunkFinish:
				events <- Event{Type: EventFinish}
			}
		}
	}()
	return events
}

// toolResults matches tool call IDs to their outcomes, delivered via goai's
// OnToolCall hook, and hands them to the pump in call order.
type toolResults struct {
	mu     sync.Mutex
	cond   *sync.Cond
	byID   map[string]goai.ToolCallInfo
	closed bool
}

func newToolResults() *toolResults {
	t := &toolResults{byID: make(map[string]goai.ToolCallInfo)}
	t.cond = sync.NewCond(&t.mu)
	return t
}

// store is registered as goai's OnToolCall hook.
func (t *toolResults) store(info goai.ToolCallInfo) {
	t.mu.Lock()
	t.byID[info.ToolCallID] = info
	t.cond.Broadcast()
	t.mu.Unlock()
}

// wait blocks until the outcome for id is available or the stream is closed.
func (t *toolResults) wait(id string) (goai.ToolCallInfo, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for {
		if info, ok := t.byID[id]; ok {
			return info, true
		}
		if t.closed {
			return goai.ToolCallInfo{}, false
		}
		t.cond.Wait()
	}
}

// close wakes all waiters. Idempotent.
func (t *toolResults) close() {
	t.mu.Lock()
	t.closed = true
	t.cond.Broadcast()
	t.mu.Unlock()
}
