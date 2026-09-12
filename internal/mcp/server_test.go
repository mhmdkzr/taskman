package mcp

import (
	"path/filepath"
	"testing"

	"uuid"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return st
}

// TestServerToolsCallOverInMemoryTransport drives the full MCP server over an
// in-memory transport and calls tools whose schemas must accept taskman's
// uuid-as-string wire format (regression test for uuid.UUID being inferred as
// an array of integers).
func TestServerToolsCallOverInMemoryTransport(t *testing.T) {
	st := newTestStore(t)
	server := NewServer(st, git.NewClient(t.TempDir()))

	clientTransport, serverTransport := gosdkmcp.NewInMemoryTransports()
	go func() {
		_ = server.Run(t.Context(), serverTransport)
	}()

	client := gosdkmcp.NewClient(&gosdkmcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	res, err := session.CallTool(t.Context(), &gosdkmcp.CallToolParams{
		Name:      "task_create",
		Arguments: map[string]any{"title": "t", "description": "d"},
	})
	if err != nil {
		t.Fatalf("task_create: %v", err)
	}
	if res.IsError {
		t.Fatalf("task_create IsError: %v", res.Content)
	}
	created, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("task_create StructuredContent = %T, want a map", res.StructuredContent)
	}
	createdTask, ok := created["task"].(map[string]any)
	if !ok {
		t.Fatalf("task_create task = %T, want an object", created["task"])
	}
	id, ok := createdTask["id"].(string)
	if !ok {
		t.Fatalf("task_create id = %v, want a string", createdTask["id"])
	}
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("task_create id %q is not a uuid: %v", id, err)
	}
	if created["state"] != "specify" {
		t.Fatalf("task_create state = %v, want %q", created["state"], "specify")
	}
	instruction, ok := created["instruction"].(map[string]any)
	if !ok || instruction["state"] != "specify" || instruction["action"] != "dispatch" {
		t.Fatalf("task_create instruction = %v, want {state: specify, action: dispatch}", created["instruction"])
	}

	res, err = session.CallTool(t.Context(), &gosdkmcp.CallToolParams{
		Name:      "task_get",
		Arguments: map[string]any{"id": id},
	})
	if err != nil {
		t.Fatalf("task_get: %v", err)
	}
	if res.IsError {
		t.Fatalf("task_get IsError: %v", res.Content)
	}
	got, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("task_get StructuredContent = %T, want a map", res.StructuredContent)
	}
	gotTask, ok := got["task"].(map[string]any)
	if !ok {
		t.Fatalf("task_get task = %T, want an object", got["task"])
	}
	if gotTask["id"] != id {
		t.Fatalf("task_get id = %v, want %s", gotTask["id"], id)
	}
	if got["state"] != "specify" {
		t.Fatalf("task_get state = %v, want %q", got["state"], "specify")
	}
}
