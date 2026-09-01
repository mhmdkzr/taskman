package spawn

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/zendev-sh/goai"
)

type spawnRecorder struct {
	mu       sync.Mutex
	subjects []string
	msgs     map[string]json.RawMessage
}

func startSpawnRecorder(t *testing.T) (*spawnRecorder, publisher.Publisher) {
	t.Helper()
	pub, err := publisher.ConnectOn(0)
	if err != nil {
		t.Fatalf("publisher.Connect: %v", err)
	}
	r := &spawnRecorder{msgs: make(map[string]json.RawMessage)}
	sub, err := pub.Subscribe(context.Background(), "agent.>", "spawn-recorder", func(m publisher.Message) {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.subjects = append(r.subjects, m.Subject())
		r.msgs[m.Subject()] = append(json.RawMessage(nil), m.Data()...)
		_ = m.Ack()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	t.Cleanup(sub.Stop)
	return r, pub
}

func (r *spawnRecorder) waitFor(t *testing.T, subject string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		r.mu.Lock()
		_, ok := r.msgs[subject]
		r.mu.Unlock()
		if ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %q; recorded: %v", subject, r.subjects)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (r *spawnRecorder) decode(t *testing.T, subject string, out any) {
	t.Helper()
	r.mu.Lock()
	data, ok := r.msgs[subject]
	r.mu.Unlock()
	if !ok {
		t.Fatalf("no message recorded for %q", subject)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal %q: %v", subject, err)
	}
}

func runnerWithPublisher(st *store.Store, pub publisher.Publisher, exec Executor) *Runner {
	return NewRunner(st, ExecOptions{
		Model:           "test-model",
		ReasoningEffort: "medium",
		MaxSteps:        10,
		SystemPrompt:    "you are a test",
	}, pub, []goai.Tool{testClockTool()}, exec, 3)
}

// TestSubagentLifecycleEvents verifies spawn publishes subagent.spawned and,
// once the run is persisted, subagent.finished.
func TestSubagentLifecycleEvents(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	rec, pub := startSpawnRecorder(t)
	r := runnerWithPublisher(st, pub, func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		return answerResult("done"), nil
	})

	out, err := r.SpawnTool(1).Execute(ctx, mustJSON(t, spawnInput{Prompt: "task"}))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	sessionID := parseSessionID(t, out)

	rec.waitFor(t, "agent.subagent.spawned", 5*time.Second)
	var spawned events.SubagentSpawned
	rec.decode(t, "agent.subagent.spawned", &spawned)
	if spawned.SessionID != sessionID || spawned.Depth != 1 || spawned.Prompt != "task" {
		t.Errorf("SubagentSpawned = %+v", spawned)
	}

	pollResult(t, ctx, r.ResultTool(), sessionID, 5*time.Second)
	rec.waitFor(t, "agent.subagent.finished", 5*time.Second)
	var finished events.SubagentFinished
	rec.decode(t, "agent.subagent.finished", &finished)
	if finished.SessionID != sessionID || finished.Text != "done" || finished.Steps != 1 {
		t.Errorf("SubagentFinished = %+v", finished)
	}
	if finished.Usage.TotalTokens != 100 {
		t.Errorf("SubagentFinished.Usage = %+v", finished.Usage)
	}
}

// TestSubagentFailedEvent verifies a failing executor publishes
// subagent.failed.
func TestSubagentFailedEvent(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	rec, pub := startSpawnRecorder(t)
	r := runnerWithPublisher(st, pub, func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		return nil, errors.New("boom")
	})

	out, err := r.SpawnTool(1).Execute(ctx, mustJSON(t, spawnInput{Prompt: "do it"}))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	sessionID := parseSessionID(t, out)

	pollResult(t, ctx, r.ResultTool(), sessionID, 5*time.Second)
	rec.waitFor(t, "agent.subagent.failed", 5*time.Second)
	var failed events.SubagentFailed
	rec.decode(t, "agent.subagent.failed", &failed)
	if failed.SessionID != sessionID || failed.Prompt != "do it" || failed.Error != "boom" {
		t.Errorf("SubagentFailed = %+v", failed)
	}
}
