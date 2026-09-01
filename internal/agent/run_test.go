package agent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/zendev-sh/goai"
)

// TestRunWrapper verifies the package-level Run helper executes a turn through
// an injected fake model.
func TestRunWrapper(t *testing.T) {
	ctx := context.Background()
	opts := testOptions([]goai.Tool{testClockTool()}, &fakeModel{})
	res, err := Run(ctx, opts, publisher.Publisher{}, "hello", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Text != "final answer" {
		t.Errorf("text = %q", res.Text)
	}
}

// TestStreamWrapper verifies the package-level Stream helper starts a turn.
func TestStreamWrapper(t *testing.T) {
	ctx := context.Background()
	opts := testOptions([]goai.Tool{testClockTool()}, &streamModel{text: "streamed"})
	stream, err := Stream(ctx, opts, publisher.Publisher{}, "hello")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	var text string
	for ev := range stream.Events() {
		if ev.Type == EventText {
			text += ev.Text
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream.Err: %v", err)
	}
	if text != "streamed" {
		t.Errorf("text = %q", text)
	}
}

// TestDefaultTools verifies the default tool set is assembled: base tools plus
// the spawn/result tools, with telegram omitted when unconfigured.
func TestDefaultTools(t *testing.T) {
	st := newTestStore(t)
	o := Options{Config: config.AgentConfig{DBPath: "taskman-test.db"}, Model: "m"}
	tools, runner, err := DefaultTools(st, publisher.Publisher{}, o)
	if err != nil {
		t.Fatalf("DefaultTools: %v", err)
	}
	if runner == nil {
		t.Fatal("nil runner")
	}
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, want := range []string{
		"grep", "glob", "read", "edit",
		"go_build", "go_test",
		"task_create", "task_search", "task_get", "task_edit",
		"spawn_subagent", "subagent_result",
	} {
		if !names[want] {
			t.Errorf("default tools missing %q: %v", want, names)
		}
	}
	if names["telegram_send"] || names["telegram_read"] {
		t.Errorf("unexpected credential-gated tools: %v", names)
	}
	// gofmt/goimports/go vet/staticcheck/golangci-lint are deterministic and
	// run automatically as pipeline steps, not as agent-invoked tools.
	for _, notWant := range []string{"go_vet", "go_fmt", "go_imports"} {
		if names[notWant] {
			t.Errorf("default tools should not include %q: %v", notWant, names)
		}
	}
}

// TestDefaultOptions verifies DefaultOptions applies defaults (model, reasoning
// effort, max steps, db path).
func TestDefaultOptions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	opts, err := DefaultOptions()
	if err != nil {
		t.Fatalf("DefaultOptions: %v", err)
	}
	if opts.Model != DefaultModel || opts.ReasoningEffort != DefaultReasoningEffort || opts.MaxSteps != DefaultMaxSteps {
		t.Errorf("defaults = %+v", opts)
	}
	if opts.Config.DBPath != filepath.Join(home, ".taskman", "taskman.db") {
		t.Errorf("db path = %q", opts.Config.DBPath)
	}
}
