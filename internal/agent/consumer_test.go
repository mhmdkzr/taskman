package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/store"
)

func TestMeaningfulOverride(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"default", ""},
		{"DEFAULT", ""},
		{"  default  ", ""},
		{"deepseek-v4-flash", "deepseek-v4-flash"},
		{"low", "low"},
	}
	for _, tc := range cases {
		if got := MeaningfulOverride(tc.in); got != tc.want {
			t.Errorf("MeaningfulOverride(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestConsumerResolveIgnoresDefaultSentinels(t *testing.T) {
	c := &Consumer{
		base: Options{
			Model:           "base-model",
			ReasoningEffort: ReasoningEffortMedium,
			MaxSteps:        100,
			SystemPrompt:    "base prompt",
		},
	}

	got := c.resolve(RunRequest{Prompt: "hi", Model: "default", ReasoningEffort: "default", SystemPrompt: "default"})
	if got.Model != "base-model" || got.ReasoningEffort != ReasoningEffortMedium || got.SystemPrompt != "base prompt" {
		t.Errorf("resolve kept sentinel overrides: %+v", got)
	}

	got = c.resolve(RunRequest{Prompt: "hi", Model: "m2", ReasoningEffort: "low", SystemPrompt: "override", MaxSteps: 5})
	if got.Model != "m2" || got.ReasoningEffort != ReasoningEffortLow || got.SystemPrompt != "override" || got.MaxSteps != 5 {
		t.Errorf("resolve dropped real overrides: %+v", got)
	}
}

func consumerOpts() Options {
	return testOptions(nil, &fakeModel{})
}

// startConsumer runs cons.Run until ctx is cancelled and blocks until the
// subscription is live. It returns a channel the caller can receive the Run
// error from.
func startConsumer(t *testing.T, ctx context.Context, cons *Consumer) chan error {
	t.Helper()
	ch := make(chan error, 1)
	go func() { ch <- cons.Run(ctx) }()
	select {
	case <-cons.Ready():
		return ch
	case <-time.After(5 * time.Second):
		t.Fatal("consumer never became ready")
		return nil
	}
}

// awaitRun waits for the consumer's Run goroutine to return and reports its
// error, if any.
func awaitRun(t *testing.T, ch chan error) {
	t.Helper()
	select {
	case err := <-ch:
		if err != nil {
			t.Errorf("consumer Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("consumer Run did not return after cancel")
	}
}

func TestConsumerRunsFreshSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := newTestStore(t)
	rec, pub := startRecorder(t)

	cons, err := NewConsumer(st, consumerOpts(), pub, 1)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	runErr := startConsumer(t, ctx, cons)

	data, err := json.Marshal(RunRequest{Prompt: "scheduled prompt"})
	if err != nil {
		t.Fatal(err)
	}
	if err := pub.PublishMsg(ctx, RunSubject, data, "test-run-1"); err != nil {
		t.Fatalf("PublishMsg: %v", err)
	}

	rec.waitFor(t, "agent.run.started", 5*time.Second)
	rec.waitFor(t, "agent.run.finished", 5*time.Second)

	var finished events.AgentRunFinished
	rec.decode(t, "agent.run.finished", &finished)
	if finished.Prompt != "scheduled prompt" || finished.Text != "final answer" {
		t.Errorf("AgentRunFinished = %+v", finished)
	}
	if finished.SessionID == "" {
		t.Fatal("no session id recorded")
	}

	tr, err := store.GetSessionTranscript(ctx, st.RO(), finished.SessionID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 1 {
		t.Errorf("turns = %d, want 1", len(tr.Turns))
	}

	cancel()
	if err := cons.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	awaitRun(t, runErr)
}

func TestConsumerContinuesSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := newTestStore(t)
	sessionID := seedSession(t, st, 1)
	rec, pub := startRecorder(t)

	cons, err := NewConsumer(st, consumerOpts(), pub, 1)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	runErr := startConsumer(t, ctx, cons)

	data, err := json.Marshal(RunRequest{Prompt: "another turn", SessionID: sessionID})
	if err != nil {
		t.Fatal(err)
	}
	if err := pub.PublishMsg(ctx, RunSubject, data, "test-run-2"); err != nil {
		t.Fatalf("PublishMsg: %v", err)
	}

	rec.waitFor(t, "agent.run.finished", 5*time.Second)

	var finished events.AgentRunFinished
	rec.decode(t, "agent.run.finished", &finished)
	if finished.SessionID != sessionID {
		t.Errorf("session id = %q, want %q", finished.SessionID, sessionID)
	}

	tr, err := store.GetSessionTranscript(ctx, st.RO(), sessionID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 2 {
		t.Errorf("turns = %d, want 2", len(tr.Turns))
	}

	cancel()
	if err := cons.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	awaitRun(t, runErr)
}

func TestConsumerRejectsBadMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := newTestStore(t)
	rec, pub := startRecorder(t)

	cons, err := NewConsumer(st, consumerOpts(), pub, 1)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	runErr := startConsumer(t, ctx, cons)

	if err := pub.PublishMsg(ctx, RunSubject, []byte("not json"), "test-run-3"); err != nil {
		t.Fatalf("PublishMsg: %v", err)
	}

	rec.waitFor(t, "agent.run.failed", 5*time.Second)
	var failed events.AgentRunFailed
	rec.decode(t, "agent.run.failed", &failed)
	if failed.Error == "" {
		t.Error("AgentRunFailed has no error")
	}

	cancel()
	if err := cons.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	awaitRun(t, runErr)
}
