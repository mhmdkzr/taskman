package spawn

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/usage"
	"github.com/zendev-sh/goai"
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

func newTestRunner(st *store.Store, exec Executor, maxDepth int) *Runner {
	return NewRunner(st, ExecOptions{
		Model:           "test-model",
		ReasoningEffort: "medium",
		MaxSteps:        10,
		SystemPrompt:    "you are a test",
	}, publisher.Publisher{}, []goai.Tool{testClockTool()}, exec, maxDepth)
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func parseSessionID(t *testing.T, out string) string {
	t.Helper()
	i := strings.Index(out, "session ")
	if i < 0 {
		t.Fatalf("no session id in output: %q", out)
	}
	rest := out[i+len("session "):]
	if j := strings.IndexByte(rest, ' '); j >= 0 {
		rest = rest[:j]
	}
	if rest == "" {
		t.Fatalf("empty session id in output: %q", out)
	}
	return rest
}

func answerResult(answer string) *ExecResult {
	return &ExecResult{
		Text:  answer,
		Usage: usage.TokenUsage{TotalTokens: 100},
		Messages: []store.Message{{
			Role: "assistant", Number: 1, FinishReason: "stop",
			Parts: []store.Part{{Type: "text", Text: answer}},
		}},
	}
}

// TestAsyncSpawnResult covers the core flow: spawn returns a session id
// immediately, the sub-agent reports running until its executor finishes, and
// subagent_result then returns its answer with the run persisted as a turn.
func TestAsyncSpawnResult(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	started := make(chan struct{})
	release := make(chan struct{})
	var got ExecOptions
	exec := func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		got = opts
		close(started)
		<-release
		return answerResult("42 is the answer"), nil
	}
	r := newTestRunner(st, exec, 3)
	spawnTool := r.SpawnTool(1)
	resultTool := r.ResultTool()

	out, err := spawnTool.Execute(ctx, mustJSON(t, spawnInput{Prompt: "compute the answer"}))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	sessionID := parseSessionID(t, out)

	<-started

	// The executor resolved options from the runner's defaults.
	if got.Model != "test-model" || got.ReasoningEffort != "medium" || got.MaxSteps != 10 || got.SystemPrompt != "you are a test" {
		t.Errorf("exec options = %+v", got)
	}
	if got.Prompt != "compute the answer" {
		t.Errorf("prompt = %q", got.Prompt)
	}
	if len(got.Tools) != 3 {
		t.Fatalf("child tools = %d, want 3", len(got.Tools))
	}
	names := map[string]bool{}
	for _, tool := range got.Tools {
		names[tool.Name] = true
	}
	if !names[SpawnToolName] || !names[ResultToolName] {
		t.Errorf("child tools missing spawn/result: %v", names)
	}

	rout, err := resultTool.Execute(ctx, mustJSON(t, resultInput{SessionID: sessionID}))
	if err != nil {
		t.Fatalf("result while running: %v", err)
	}
	if !strings.Contains(rout, "still running") {
		t.Errorf("running status = %q, want 'still running'", rout)
	}

	close(release)
	rout = pollResult(t, ctx, resultTool, sessionID, 5*time.Second)
	if !strings.Contains(rout, "42 is the answer") {
		t.Errorf("result missing answer: %q", rout)
	}

	// The run is persisted as a turn of the sub-agent's session, and the
	// session's tool set includes the spawn tool.
	tr, err := store.GetSessionTranscript(ctx, st.RW(), sessionID)
	if err != nil {
		t.Fatalf("transcript: %v", err)
	}
	if len(tr.Turns) != 1 {
		t.Fatalf("turns = %d, want 1", len(tr.Turns))
	}
	if tr.Session.Model != "test-model" {
		t.Errorf("session model = %q", tr.Session.Model)
	}
	foundSpawn := false
	for _, tool := range tr.Session.Tools {
		if tool.Name == SpawnToolName {
			foundSpawn = true
		}
	}
	if !foundSpawn {
		t.Errorf("session tools missing %s: %+v", SpawnToolName, tr.Session.Tools)
	}
}

// TestSpawnError verifies a failing executor is surfaced by subagent_result as
// a failure, not as a still-running sub-agent.
func TestSpawnError(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	exec := func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		return nil, errors.New("boom")
	}
	r := newTestRunner(st, exec, 3)

	out, err := r.SpawnTool(1).Execute(ctx, mustJSON(t, spawnInput{Prompt: "do it"}))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	sessionID := parseSessionID(t, out)

	rout := pollResult(t, ctx, r.ResultTool(), sessionID, 5*time.Second)
	if !strings.Contains(rout, "failed") || !strings.Contains(rout, "boom") {
		t.Errorf("error result = %q, want failure with 'boom'", rout)
	}
}

// TestSpawnWait verifies wait=true blocks until the sub-agent finishes and
// returns its answer inline.
func TestSpawnWait(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	release := make(chan struct{})
	exec := func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		<-release
		return answerResult("waited answer"), nil
	}
	r := newTestRunner(st, exec, 3)

	go func() {
		time.Sleep(100 * time.Millisecond)
		close(release)
	}()

	out, err := r.SpawnTool(1).Execute(ctx, mustJSON(t, spawnInput{Prompt: "do it", Wait: boolPtr(true)}))
	if err != nil {
		t.Fatalf("spawn wait: %v", err)
	}
	if !strings.Contains(out, "waited answer") {
		t.Errorf("wait output = %q, want answer", out)
	}
	if !strings.Contains(out, "(status: done)") {
		t.Errorf("wait output = %q, want done status", out)
	}
}

// TestResultWaitSeconds verifies subagent_result blocks up to wait_seconds for
// completion.
func TestResultWaitSeconds(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	started := make(chan struct{})
	release := make(chan struct{})
	exec := func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		close(started)
		<-release
		return answerResult("eventual answer"), nil
	}
	r := newTestRunner(st, exec, 3)

	out, err := r.SpawnTool(1).Execute(ctx, mustJSON(t, spawnInput{Prompt: "do it"}))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	sessionID := parseSessionID(t, out)
	<-started

	go func() {
		time.Sleep(100 * time.Millisecond)
		close(release)
	}()

	start := time.Now()
	rout, err := r.ResultTool().Execute(ctx, mustJSON(t, resultInput{SessionID: sessionID, WaitSeconds: intPtr(10)}))
	if err != nil {
		t.Fatalf("result wait: %v", err)
	}
	if !strings.Contains(rout, "eventual answer") {
		t.Errorf("wait result = %q", rout)
	}
	if time.Since(start) < 50*time.Millisecond {
		t.Errorf("wait returned too fast: %v", time.Since(start))
	}
}

// TestDepthLimit verifies spawning past maxDepth fails with a tool error.
func TestDepthLimit(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	r := newTestRunner(st, func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		return answerResult("ok"), nil
	}, 2)

	// Children at depth 2 are allowed; depth 3 exceeds the limit.
	if _, err := r.SpawnTool(2).Execute(ctx, mustJSON(t, spawnInput{Prompt: "fine"})); err != nil {
		t.Fatalf("spawn at depth 2: %v", err)
	}
	if _, err := r.SpawnTool(3).Execute(ctx, mustJSON(t, spawnInput{Prompt: "too deep"})); err == nil {
		t.Fatal("spawn at depth 3: want depth error")
	} else if !strings.Contains(err.Error(), "maximum sub-agent depth") {
		t.Errorf("depth error = %v", err)
	}
}

// TestRunnerWait verifies Wait blocks until every in-flight sub-agent has
// finished.
func TestRunnerWait(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	release := make(chan struct{})
	started := make(chan struct{}, 3)
	exec := func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		started <- struct{}{}
		<-release
		return answerResult("done"), nil
	}
	r := newTestRunner(st, exec, 3)

	for i := 0; i < 3; i++ {
		if _, err := r.SpawnTool(1).Execute(ctx, mustJSON(t, spawnInput{Prompt: "task"})); err != nil {
			t.Fatalf("spawn %d: %v", i, err)
		}
	}
	for i := 0; i < 3; i++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("sub-agent did not start")
		}
	}

	waited := make(chan error, 1)
	go func() { waited <- r.Wait(ctx) }()

	select {
	case err := <-waited:
		t.Fatalf("Wait returned while sub-agents in flight: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	close(release)

	select {
	case err := <-waited:
		if err != nil {
			t.Fatalf("Wait: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wait did not return after release")
	}

	// A second Wait on an empty runner returns immediately.
	if err := r.Wait(ctx); err != nil {
		t.Fatalf("Wait on idle runner: %v", err)
	}
}

// TestSpawnValidation covers bad input: missing prompt and empty session id.
func TestSpawnValidation(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	r := newTestRunner(st, func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		return answerResult("ok"), nil
	}, 3)

	if _, err := r.SpawnTool(1).Execute(ctx, mustJSON(t, spawnInput{})); err == nil {
		t.Error("spawn with empty prompt: want error")
	}
	if _, err := r.ResultTool().Execute(ctx, mustJSON(t, resultInput{})); err == nil {
		t.Error("result with empty session_id: want error")
	}
	if _, err := r.ResultTool().Execute(ctx, mustJSON(t, resultInput{SessionID: "missing", WaitSeconds: intPtr(-1)})); err == nil {
		t.Error("result with negative wait_seconds: want error")
	}
}

func pollResult(t *testing.T, ctx context.Context, tool goai.Tool, sessionID string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		out, err := tool.Execute(ctx, mustJSON(t, resultInput{SessionID: sessionID}))
		if err != nil {
			t.Fatalf("result: %v", err)
		}
		if !strings.Contains(out, "still running") {
			return out
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for result; last status: %q", out)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }
