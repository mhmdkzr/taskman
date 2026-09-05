package migrations

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSchemaApply(t *testing.T) {
	schema, err := migrationsFS.ReadFile("schema.sql")
	if err != nil {
		t.Fatalf("read embedded schema: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "migrations.sqlite")
	apply := exec.Command("sqlite3", "-bail", dbPath)
	apply.Stdin = bytes.NewReader(schema)
	if output, err := apply.CombinedOutput(); err != nil {
		t.Fatalf("apply schema: %v\n%s", err, output)
	}

	tables := []string{
		"agent_sessions", "agent_tools", "agents", "model_providers", "models",
		"prompt_templates", "session_todos", "session_tools", "session_turns", "tools",
	}

	list := exec.Command("sqlite3", "-noheader", dbPath,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name;")
	output, err := list.Output()
	if err != nil {
		t.Fatalf("list schema tables: %v", err)
	}
	got := strings.Fields(string(output))
	if !slices.Equal(got, tables) {
		t.Fatalf("tables = %v, want %v", got, tables)
	}

	assertRejected(
		t,
		dbPath,
		"INSERT INTO tools (tool_id, tool_name, tool_description, input_schema, output_schema) VALUES ('tool', 'name', 'description', 'not json', '{}');",
	)
}

func assertRejected(t *testing.T, dbPath, statement string) {
	t.Helper()

	command := exec.Command("sqlite3", "-bail", dbPath, statement)
	if output, err := command.CombinedOutput(); err == nil {
		t.Fatalf("statement unexpectedly succeeded: %s\n%s", statement, output)
	}
}
