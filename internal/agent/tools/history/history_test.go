package history

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/store"
	"github.com/mhmdkzr/loop/migrations"
	"github.com/mhmdkzr/loop/pkg/migrate"
	"github.com/mhmdkzr/loop/pkg/testenv"
)

// openTestStore opens a migrated Store backed by a fresh temp-file database.
func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "history.sqlite"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	if err := migrate.Migrate(t.Context(), st.RW(), migrations.GetMigrationsFS()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return st
}

// seedTurn inserts a full session (with its provider, model, prompt
// template, and agent rows) plus a single turn, and returns the session id.
func seedTurn(t *testing.T, db *sql.DB, prompt, reply string) sessions.SessionID {
	t.Helper()
	ctx := t.Context()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	providerID := uuid.NewV7()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO model_providers (provider_id, provider_name) VALUES (?, ?)`,
		providerID.String(), "test-provider-"+providerID.String()); err != nil {
		t.Fatalf("insert provider: %v", err)
	}

	modelID := uuid.NewV7()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO models (model_id, provider_id, model_name, context_window, has_vision) VALUES (?, ?, ?, 0, 0)`,
		modelID.String(), providerID.String(), "test-model"); err != nil {
		t.Fatalf("insert model: %v", err)
	}

	promptID := uuid.NewV7()
	promptQuery := `INSERT INTO prompt_templates (prompt_id, prompt_name, template_body, params_schema, version) VALUES (?, ?, ?, ?, 1)`
	if _, err := db.ExecContext(ctx, promptQuery,
		promptID.String(), "test-prompt-"+promptID.String(), "you are a test agent", "{}"); err != nil {
		t.Fatalf("insert prompt template: %v", err)
	}

	agentID := uuid.NewV7()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO agents (agent_id, agent_name, prompt_id, model_id) VALUES (?, ?, ?, ?)`,
		agentID.String(), "test-agent-"+agentID.String(), promptID.String(), modelID.String()); err != nil {
		t.Fatalf("insert agent: %v", err)
	}

	sessionID := uuid.NewV7()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO agent_sessions (session_id, agent_id, model_id, system_prompt, created_at) VALUES (?, ?, ?, ?, ?)`,
		sessionID.String(), agentID.String(), modelID.String(), "you are a test agent", now); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	result, err := json.Marshal(goai.TextResult{Text: reply})
	if err != nil {
		t.Fatalf("marshal reply: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO session_turns (turn_id, session_id, prompt, result, created_at) VALUES (?, ?, ?, ?, ?)`,
		uuid.NewV7().String(), sessionID.String(), prompt, string(result), now); err != nil {
		t.Fatalf("insert turn: %v", err)
	}

	return sessions.SessionID(sessionID)
}

func TestDBExecuteReturnsRecentTurns(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	id := seedTurn(t, st.RW(), "where did we leave off?", "we were fixing the reporting bug")

	out, err := execute(t.Context(), st, Input{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	for _, want := range []string{
		"session " + id.String(),
		"where did we leave off?",
		"we were fixing the reporting bug",
	} {
		if !strings.Contains(out.History, want) {
			t.Errorf("history missing %q:\n%s", want, out.History)
		}
	}
}

func TestDBExecuteFiltersByQuery(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	seedTurn(t, st.RW(), "where did we leave off?", "we were fixing the reporting bug")
	seedTurn(t, st.RW(), "summarize the deploy", "deploy is on hold until tests pass")

	out, err := execute(t.Context(), st, Input{Query: "reporting"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.History, "where did we leave off?") {
		t.Errorf("history missing matching turn:\n%s", out.History)
	}
	if strings.Contains(out.History, "summarize the deploy") {
		t.Errorf("history contains non-matching turn:\n%s", out.History)
	}
}

func TestDBExecuteFiltersBySession(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	seedTurn(t, st.RW(), "prompt one", "reply one")
	id2 := seedTurn(t, st.RW(), "prompt two", "reply two")

	out, err := execute(t.Context(), st, Input{SessionID: id2.String()})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.History, "prompt two") {
		t.Errorf("history missing filtered session:\n%s", out.History)
	}
	if strings.Contains(out.History, "prompt one") {
		t.Errorf("history contains other session:\n%s", out.History)
	}
}

func TestDBExecuteTruncatesLongMessages(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	long := strings.Repeat("x", 500)
	seedTurn(t, st.RW(), "long message", long)

	out, err := execute(t.Context(), st, Input{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(out.History, long) {
		t.Error("long reply not truncated")
	}
	if !strings.Contains(out.History, "truncated") {
		t.Errorf("missing truncation footer:\n%s", out.History)
	}
}

func TestExecuteNilStore(t *testing.T) {
	if _, err := execute(t.Context(), nil, Input{}); err == nil {
		t.Error("expected nil-store error")
	}
}

func TestExecuteInvalidSessionID(t *testing.T) {
	st := openTestStore(t)
	if _, err := execute(t.Context(), st, Input{SessionID: "not-a-uuid"}); err == nil {
		t.Error("expected invalid session_id error")
	}
}

func TestValidateRejectsOutOfRangeLimits(t *testing.T) {
	for _, limit := range []int{0, maxLimit + 1} {
		if err := (Input{Limit: &limit}).Validate(); err == nil {
			t.Errorf("limit %d: expected validation error", limit)
		}
	}
}
