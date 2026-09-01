package events

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/publisher"
)

func TestSubjectsDistinct(t *testing.T) {
	events := []publisher.Event[any]{
		Request{}, Response{}, StepFinished{}, ToolCallStarted{}, ToolCalled{},
		GenerationFinished{}, Panic{}, TurnStarted{}, TurnFinished{},
		SubagentSpawned{}, SubagentFinished{},
		SubagentFailed{}, SchedulerExpired{}, RecurringMissed{},
	}
	seen := make(map[string]string)
	for _, e := range events {
		subj := e.Subject()
		if subj == "" {
			t.Errorf("event %T has an empty subject", e)
			continue
		}
		if prev, ok := seen[subj]; ok {
			t.Errorf("subject %q used by both %s and %T", subj, prev, e)
		}
		seen[subj] = e.Subject()
	}
}

// TestMsgIDsDeterministic verifies every event derives a non-empty, stable
// dedup id from its contents: identical contents yield the same id.
func TestMsgIDsDeterministic(t *testing.T) {
	events := []publisher.Event[any]{
		Request{Model: "m", Timestamp: "t"},
		Response{Latency: "1s", StatusCode: 200},
		StepFinished{Number: 1, Text: "t"},
		ToolCallStarted{ID: "tc-1", Name: "bash", Step: 2},
		ToolCalled{ID: "tc-1", Name: "bash", Step: 2, Output: "out"},
		GenerationFinished{TotalSteps: 3},
		Panic{Phase: "tool", Value: "boom"},
		TurnStarted{SessionID: "s1", Prompt: "hi"},
		TurnFinished{SessionID: "s1", Prompt: "hi", Text: "ok"},
		SubagentSpawned{SessionID: "s3"},
		SubagentFinished{SessionID: "s3"},
		SubagentFailed{SessionID: "s3", Error: "e"},
		SchedulerExpired{ID: "x"},
		RecurringMissed{RecurrenceID: "r1", OccurredAt: 123},
	}
	for _, e := range events {
		if e.MsgID() == "" {
			t.Errorf("event %T has an empty msg id", e)
		}
		if got, want := e.MsgID(), e.MsgID(); got != want {
			t.Errorf("event %T msg id is not stable: %q vs %q", e, got, want)
		}
	}
}

// TestMsgIDsDistinct verifies events with different contents produce different
// ids, so distinct events are never deduplicated into one.
func TestMsgIDsDistinct(t *testing.T) {
	pairs := []struct {
		a, b publisher.Event[any]
	}{
		{TurnStarted{SessionID: "s1", Prompt: "hi"}, TurnStarted{SessionID: "s1", Prompt: "bye"}},
		{TurnStarted{SessionID: "s1", Prompt: "hi"}, TurnStarted{SessionID: "s2", Prompt: "hi"}},
		{Request{Model: "m", Timestamp: "t1"}, Request{Model: "m", Timestamp: "t2"}},
		{ToolCallStarted{ID: "tc-1", Name: "bash"}, ToolCallStarted{ID: "tc-2", Name: "bash"}},
		{SubagentSpawned{SessionID: "s1"}, SubagentSpawned{SessionID: "s2"}},
		{SchedulerExpired{ID: "a"}, SchedulerExpired{ID: "b"}},
		{RecurringMissed{RecurrenceID: "a", OccurredAt: 1}, RecurringMissed{RecurrenceID: "b", OccurredAt: 1}},
		{RecurringMissed{RecurrenceID: "a", OccurredAt: 1}, RecurringMissed{RecurrenceID: "a", OccurredAt: 2}},
	}
	for _, p := range pairs {
		if p.a.MsgID() == p.b.MsgID() {
			t.Errorf("distinct events share msg id %q: %+v vs %+v", p.a.MsgID(), p.a, p.b)
		}
	}
}
