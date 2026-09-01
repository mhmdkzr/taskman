package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/mhmdkzr/taskman/internal/agent"
	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/protocol"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/zendev-sh/goai/provider"
)

// fakeModel answers directly with a fixed text, no tool calls.
type fakeModel struct{}

func (fakeModel) ModelID() string { return "fake-model" }

func (fakeModel) DoGenerate(ctx context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	return &provider.GenerateResult{
		Text:         "final answer",
		FinishReason: provider.FinishStop,
		Usage:        provider.Usage{InputTokens: 20, OutputTokens: 5, TotalTokens: 25},
	}, nil
}

func (fakeModel) DoStream(ctx context.Context, params provider.GenerateParams) (*provider.StreamResult, error) {
	return nil, nil
}

// testClient is a minimal NATS request/reply client for the protocol subject,
// standing in for the CLI client so the server tests can talk to the runtime.
type testClient struct {
	nc *nats.Conn
}

func (c *testClient) Ping(ctx context.Context) (protocol.PingReply, error) {
	var reply protocol.PingReply
	resp, err := c.request(ctx, protocol.CommandPing, nil)
	if err != nil {
		return reply, err
	}
	if err := json.Unmarshal(resp.Data, &reply); err != nil {
		return reply, fmt.Errorf("decode ping reply: %w", err)
	}
	return reply, nil
}

func (c *testClient) Run(ctx context.Context, req protocol.RunRequest) (protocol.RunReply, error) {
	var reply protocol.RunReply
	resp, err := c.request(ctx, protocol.CommandRun, req)
	if err != nil {
		return reply, err
	}
	if err := json.Unmarshal(resp.Data, &reply); err != nil {
		return reply, fmt.Errorf("decode run reply: %w", err)
	}
	return reply, nil
}

func (c *testClient) request(ctx context.Context, command string, data any) (*protocol.Reply, error) {
	var raw json.RawMessage
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshal %s request: %w", command, err)
		}
		raw = b
	}
	payload, err := json.Marshal(protocol.Request{Command: command, Data: raw})
	if err != nil {
		return nil, err
	}
	msg, err := c.nc.RequestWithContext(ctx, protocol.Subject, payload)
	if err != nil {
		if errors.Is(err, nats.ErrNoResponders) {
			return nil, errors.New("no server is running")
		}
		return nil, fmt.Errorf("%s request: %w", command, err)
	}
	var reply protocol.Reply
	if err := json.Unmarshal(msg.Data, &reply); err != nil {
		return nil, fmt.Errorf("decode %s reply: %w", command, err)
	}
	if !reply.OK {
		if reply.Error == "" {
			reply.Error = "unknown error"
		}
		return nil, fmt.Errorf("%s: %s", command, reply.Error)
	}
	return &reply, nil
}

// newTestServer starts an embedded bus, a server on it with a fake model, and
// a client connected to the same bus. It returns once the server's request
// subscription is live (a ping succeeds).
func newTestServer(t *testing.T) (*Server, *testClient) {
	t.Helper()
	pub, err := publisher.ConnectOn(0)
	if err != nil {
		t.Fatalf("ConnectOn: %v", err)
	}

	opts := agent.Options{
		Config: config.AgentConfig{DBPath: filepath.Join(t.TempDir(), "taskman.db")},
		Model:  "fake-model",
	}
	opts.SetModelForTesting(&fakeModel{})

	srv, err := New(opts, pub)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { srv.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx, log.New(io.Discard, "", 0)) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("server Run: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("server did not stop")
		}
	})

	c := &testClient{nc: pub.Conn()}

	deadline := time.Now().Add(5 * time.Second)
	for {
		pingCtx, pingCancel := context.WithTimeout(ctx, time.Second)
		_, pingErr := c.Ping(pingCtx)
		pingCancel()
		if pingErr == nil {
			return srv, c
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never became ready: %v", pingErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestPing(t *testing.T) {
	_, c := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reply, err := c.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if reply.Version == "" {
		t.Error("empty version")
	}
	if reply.Uptime == "" {
		t.Error("empty uptime")
	}
}

func TestRunFreshSession(t *testing.T) {
	srv, c := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reply, err := c.Run(ctx, protocol.RunRequest{Prompt: "hello"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if reply.Text != "final answer" {
		t.Errorf("text = %q", reply.Text)
	}
	if reply.SessionID == "" {
		t.Fatal("empty session id")
	}

	tr, err := store.GetSessionTranscript(ctx, srv.st.RO(), reply.SessionID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 1 {
		t.Errorf("turns = %d, want 1", len(tr.Turns))
	}
}

func TestRunContinueSession(t *testing.T) {
	srv, c := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	first, err := c.Run(ctx, protocol.RunRequest{Prompt: "first"})
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}
	second, err := c.Run(ctx, protocol.RunRequest{Prompt: "second", SessionID: first.SessionID})
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if second.SessionID != first.SessionID {
		t.Errorf("session id = %q, want %q", second.SessionID, first.SessionID)
	}
	if second.Text != "final answer" {
		t.Errorf("text = %q", second.Text)
	}

	tr, err := store.GetSessionTranscript(ctx, srv.st.RO(), first.SessionID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 2 {
		t.Errorf("turns = %d, want 2", len(tr.Turns))
	}
}

func TestRunJSON(t *testing.T) {
	_, c := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reply, err := c.Run(ctx, protocol.RunRequest{Prompt: "hello", AsJSON: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(reply.RunOutput) == 0 {
		t.Fatal("empty run output")
	}
	var out map[string]any
	if err := json.Unmarshal(reply.RunOutput, &out); err != nil {
		t.Fatalf("run output is not JSON: %v", err)
	}
	if out["Prompt"] != nil {
		t.Errorf("Prompt = %v, want absent (it lives in the transcript)", out["Prompt"])
	}
	sessionID, ok := out["SessionID"].(string)
	if !ok || sessionID == "" {
		t.Errorf("SessionID = %v, want a non-empty id", out["SessionID"])
	}
	if out["SessionID"] != reply.SessionID {
		t.Errorf("SessionID = %v, want %v", out["SessionID"], reply.SessionID)
	}
}

func TestRunRejectsInvalidRequest(t *testing.T) {
	_, c := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.Run(ctx, protocol.RunRequest{Prompt: "hi", ForkFrom: "s1"})
	if err == nil {
		t.Fatal("Run with fork but no turn: want error, got nil")
	}
	if !strings.Contains(err.Error(), "turn is required with fork") {
		t.Errorf("error = %q, want validation message", err)
	}
}
