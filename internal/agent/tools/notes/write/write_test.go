package write

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/store"
	"github.com/mhmdkzr/loop/migrations"
	"github.com/mhmdkzr/loop/pkg/migrate"
	"github.com/mhmdkzr/loop/pkg/testenv"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		in   Input
	}{
		{name: "missing name", in: Input{Body: "body"}},
		{name: "missing body", in: Input{Name: "name"}},
		{name: "missing link name", in: Input{Name: "name", Body: "body", Links: []Link{{}}}},
		{name: "self link", in: Input{Name: "name", Body: "body", Links: []Link{{Name: "name"}}}},
		{
			name: "duplicate link",
			in:   Input{Name: "name", Body: "body", Links: []Link{{Name: "other"}, {Name: "other"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.in.Validate(); err == nil {
				t.Fatal("Validate returned nil")
			}
		})
	}
}

func TestExecuteRequiresDependencies(t *testing.T) {
	if _, err := execute(t.Context(), tools.Deps{}, Input{}); err == nil {
		t.Fatal("expected database error")
	}
}

func TestDBExecuteUpsertsNoteAndReplacesLinks(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "notes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := migrate.Migrate(t.Context(), st.RW(), migrations.GetMigrationsFS()); err != nil {
		t.Fatal(err)
	}
	sessionID := seedSession(t, st.RW())
	if _, err := st.RW().ExecContext(
		t.Context(),
		`INSERT INTO notes (id, session_id, name, body, created_at) VALUES (?, ?, ?, ?, ?)`,
		uuid.NewV7(), sessionID.String(), "other", "other body", time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		t.Fatal(err)
	}

	out, err := execute(t.Context(), tools.Deps{Store: st, SessionID: sessionID}, Input{
		Name: "main", Body: "first", Links: []Link{{Name: "other", Relationship: "depends on"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "main" || out.Body != "first" || len(out.Links) != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
	if _, err := execute(t.Context(), tools.Deps{Store: st, SessionID: sessionID}, Input{
		Name: "main", Body: "second",
	}); err != nil {
		t.Fatal(err)
	}
	var body, relationship string
	if err := st.RW().QueryRowContext(
		t.Context(), `SELECT body FROM notes WHERE name = ?`, "main",
	).Scan(&body); err != nil {
		t.Fatal(err)
	}
	if body != "second" {
		t.Fatalf("body = %q, want second", body)
	}
	if err := st.RW().QueryRowContext(
		t.Context(), `SELECT relationship FROM notes_links`,
	).Scan(&relationship); err == nil {
		t.Fatal("expected old link to be removed")
	}
}

func seedSession(
	t *testing.T,
	db interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	},
) sessions.SessionID {
	t.Helper()
	ctx := t.Context()
	provider := uuid.NewV7()
	model := uuid.NewV7()
	prompt := uuid.NewV7()
	agent := uuid.NewV7()
	session := sessions.SessionID(uuid.NewV7())
	statements := []struct {
		query string
		args  []any
	}{
		{
			`INSERT INTO model_providers (provider_id, provider_name) VALUES (?, ?)`,
			[]any{provider.String(), provider.String()},
		},
		{
			`INSERT INTO models (model_id, provider_id, model_name, context_window, has_vision) VALUES (?, ?, ?, 0, 0)`,
			[]any{model.String(), provider.String(), model.String()},
		},
		{
			`INSERT INTO prompt_templates (prompt_id, prompt_name, template_body, params_schema, version) VALUES (?, ?, ?, '{}', 1)`,
			[]any{prompt.String(), prompt.String(), "prompt"},
		},
		{
			`INSERT INTO agents (agent_id, agent_name, prompt_id, model_id) VALUES (?, ?, ?, ?)`,
			[]any{agent.String(), agent.String(), prompt.String(), model.String()},
		},
		{
			`INSERT INTO agent_sessions (session_id, agent_id, model_id, system_prompt, created_at) VALUES (?, ?, ?, ?, ?)`,
			[]any{
				session.String(),
				agent.String(),
				model.String(),
				"prompt",
				time.Now().UTC().Format(time.RFC3339Nano),
			},
		},
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	return session
}
