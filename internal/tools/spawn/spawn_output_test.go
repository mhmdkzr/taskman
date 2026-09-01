package spawn

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/usage"
)

func toolCallExecResult() *ExecResult {
	return &ExecResult{
		Text:  "answer",
		Usage: usage.TokenUsage{TotalTokens: 100},
		Messages: []store.Message{
			{Role: "assistant", Number: 1, FinishReason: "tool-calls", Parts: []store.Part{
				{Type: "text", Text: "let me check"},
				{Type: "tool-call", ToolCallID: "c1", ToolName: "bash", ToolInput: `{"command":"date"}`},
			}},
			{Role: "tool", Number: 1, Parts: []store.Part{
				{Type: "tool-result", ToolCallID: "c1", ToolName: "bash", ToolOutput: "now"},
			}},
			{Role: "assistant", Number: 2, FinishReason: "stop", Parts: []store.Part{
				{Type: "text", Text: "answer"},
			}},
		},
	}
}

// TestResultIncludeSteps verifies subagent_result renders per-step narration
// and tool calls when include_steps is set.
func TestResultIncludeSteps(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	r := newTestRunner(st, func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		return toolCallExecResult(), nil
	}, 3)

	out, err := r.SpawnTool(1).Execute(ctx, mustJSON(t, spawnInput{Prompt: "task"}))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	sessionID := parseSessionID(t, out)
	pollResult(t, ctx, r.ResultTool(), sessionID, 5*time.Second)

	rout, err := r.ResultTool().Execute(ctx, mustJSON(t, resultInput{SessionID: sessionID, IncludeSteps: boolPtr(true)}))
	if err != nil {
		t.Fatalf("result: %v", err)
	}
	for _, want := range []string{"-- step 1", "-- step 2", "called bash", "-> now", "let me check"} {
		if !strings.Contains(rout, want) {
			t.Errorf("result missing %q:\n%s", want, rout)
		}
	}
}

// TestResultTruncatesLongAnswer verifies an answer longer than the output cell
// limit is truncated.
func TestResultTruncatesLongAnswer(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	long := strings.Repeat("x", defaultCellLen+100)
	r := newTestRunner(st, func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		return &ExecResult{Text: long, Usage: usage.TokenUsage{TotalTokens: 100}, Messages: []store.Message{
			{Role: "assistant", Number: 1, FinishReason: "stop", Parts: []store.Part{
				{Type: "text", Text: long},
			}},
		}}, nil
	}, 3)

	out, err := r.SpawnTool(1).Execute(ctx, mustJSON(t, spawnInput{Prompt: "task"}))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	sessionID := parseSessionID(t, out)
	pollResult(t, ctx, r.ResultTool(), sessionID, 5*time.Second)

	rout, err := r.ResultTool().Execute(ctx, mustJSON(t, resultInput{SessionID: sessionID}))
	if err != nil {
		t.Fatalf("result: %v", err)
	}
	if !strings.Contains(rout, "...") {
		t.Error("long answer was not truncated")
	}
	if len(rout) >= len(long)+200 {
		t.Error("result not shortened")
	}
}

// TestNewRunnerClampsDepth verifies a non-positive maxDepth is clamped to 1.
func TestNewRunnerClampsDepth(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	r := newTestRunner(st, func(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
		return answerResult("ok"), nil
	}, 0)
	if r.maxDepth != 1 {
		t.Fatalf("maxDepth = %d, want 1", r.maxDepth)
	}
	if _, err := r.SpawnTool(1).Execute(ctx, mustJSON(t, spawnInput{Prompt: "fine"})); err != nil {
		t.Fatalf("spawn at depth 1: %v", err)
	}
	if _, err := r.SpawnTool(2).Execute(ctx, mustJSON(t, spawnInput{Prompt: "too deep"})); err == nil {
		t.Fatal("spawn at depth 2 with clamp: want error")
	}
}
