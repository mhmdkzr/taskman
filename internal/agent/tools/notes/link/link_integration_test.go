package link

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

func TestDBExecuteUpsertsCanonicalLink(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "notes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := migrate.Migrate(t.Context(), st.RW(), migrations.GetMigrationsFS()); err != nil {
		t.Fatal(err)
	}
	sessionID := seedSession(t, st.RW())
	insertNote(t, st.RW(), sessionID, "a")
	insertNote(t, st.RW(), sessionID, "b")
	d := tools.Deps{Store: st, SessionID: sessionID}
	if _, err := execute(t.Context(), d, Input{From: "b", To: "a", Relationship: "related"}); err != nil {
		t.Fatal(err)
	}
	if _, err := execute(t.Context(), d, Input{From: "a", To: "b", Relationship: "depends on"}); err != nil {
		t.Fatal(err)
	}
	var count int
	var relationship string
	if err := st.RO().QueryRowContext(
		t.Context(), `SELECT COUNT(*), relationship FROM notes_links`,
	).Scan(&count, &relationship); err != nil {
		t.Fatal(err)
	}
	if count != 1 || relationship != "depends on" {
		t.Fatalf("link = (%d, %q)", count, relationship)
	}
}

func insertNote(
	t *testing.T,
	db interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	},
	sessionID sessions.SessionID,
	name string,
) {
	t.Helper()
	_, err := db.ExecContext(
		t.Context(),
		`INSERT INTO notes (id, session_id, name, body, created_at) VALUES (?, ?, ?, ?, ?)`,
		uuid.NewV7().String(),
		sessionID.String(),
		name,
		name+" body",
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		t.Fatal(err)
	}
}

func seedSession(
	t *testing.T,
	db interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	},
) sessions.SessionID {
	t.Helper()
	provider, model, prompt, agent := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	session := sessions.SessionID(uuid.NewV7())
	values := [][2]any{
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
	for _, value := range values {
		if _, err := db.ExecContext(t.Context(), value[0].(string), value[1].([]any)...); err != nil {
			t.Fatal(err)
		}
	}
	return session
}
