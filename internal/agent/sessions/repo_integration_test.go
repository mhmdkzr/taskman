package sessions

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/store"
	"github.com/mhmdkzr/loop/migrations"
	"github.com/mhmdkzr/loop/pkg/migrate"
	"github.com/mhmdkzr/loop/pkg/testenv"
)

// openTestStore opens a migrated Store backed by a fresh temp-file database.
func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "sessions.sqlite"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	if err := migrate.Migrate(t.Context(), st.RW(), migrations.GetMigrationsFS()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return st
}

// seedAgent inserts the provider, model, and prompt template rows a session
// needs to satisfy foreign keys, plus the agent row itself.
func seedAgent(t *testing.T, db *sql.DB) (AgentID, ModelID) {
	t.Helper()
	ctx := t.Context()

	providerID := uuid.NewV7()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO model_providers (provider_id, provider_name) VALUES (?, ?)`,
		providerID.String(), "test-provider-"+providerID.String()); err != nil {
		t.Fatalf("insert provider: %v", err)
	}

	modelID := ModelID(uuid.NewV7())
	if _, err := db.ExecContext(ctx,
		`INSERT INTO models (model_id, provider_id, model_name, context_window, has_vision) VALUES (?, ?, ?, 0, 0)`,
		modelID.String(), providerID.String(), "test-model"); err != nil {
		t.Fatalf("insert model: %v", err)
	}

	promptID := uuid.NewV7()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO prompt_templates (prompt_id, prompt_name, template_body, params_schema, version) VALUES (?, ?, ?, ?, 1)`,
		promptID.String(), "test-prompt-"+promptID.String(), "you are a test agent", "{}"); err != nil {
		t.Fatalf("insert prompt template: %v", err)
	}

	agentID := AgentID(uuid.NewV7())
	if _, err := db.ExecContext(ctx,
		`INSERT INTO agents (agent_id, agent_name, prompt_id, model_id) VALUES (?, ?, ?, ?)`,
		agentID.String(), "test-agent-"+agentID.String(), promptID.String(), modelID.String()); err != nil {
		t.Fatalf("insert agent: %v", err)
	}

	return agentID, modelID
}

func TestDBSearchHistoryOrdersNewestFirstAcrossSessions(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	agentID, modelID := seedAgent(t, st.RW())
	ctx := t.Context()

	id1, err := createSession(ctx, st.RW(), agentID, modelID, "sys", nil, nil)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if err := appendTurn(ctx, st.RW(), id1, "where did we leave off?",
		&goai.TextResult{Text: "we were fixing the reporting bug"}); err != nil {
		t.Fatalf("appendTurn: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	id2, err := createSession(ctx, st.RW(), agentID, modelID, "sys", nil, nil)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if err := appendTurn(ctx, st.RW(), id2, "summarize the deploy",
		&goai.TextResult{Text: "deploy is on hold until tests pass"}); err != nil {
		t.Fatalf("appendTurn: %v", err)
	}

	turns, err := SearchHistory(ctx, st, nil, "", 10)
	if err != nil {
		t.Fatalf("SearchHistory: %v", err)
	}
	if len(turns) != 2 {
		t.Fatalf("len(turns) = %d, want 2", len(turns))
	}
	if turns[0].SessionID != id2 || turns[1].SessionID != id1 {
		t.Errorf("turns = %+v, want newest session (%s) first", turns, id2)
	}
}

func TestDBSearchHistoryFiltersByQuery(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	agentID, modelID := seedAgent(t, st.RW())
	ctx := t.Context()

	id, err := createSession(ctx, st.RW(), agentID, modelID, "sys", nil, nil)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if err := appendTurn(ctx, st.RW(), id, "where did we leave off?",
		&goai.TextResult{Text: "we were fixing the reporting bug"}); err != nil {
		t.Fatalf("appendTurn: %v", err)
	}

	turns, err := SearchHistory(ctx, st, nil, "REPORTING", 10)
	if err != nil {
		t.Fatalf("SearchHistory: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("len(turns) = %d, want 1", len(turns))
	}

	turns, err = SearchHistory(ctx, st, nil, "no such text", 10)
	if err != nil {
		t.Fatalf("SearchHistory: %v", err)
	}
	if len(turns) != 0 {
		t.Fatalf("len(turns) = %d, want 0", len(turns))
	}
}

func TestDBSearchHistoryFiltersBySession(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	agentID, modelID := seedAgent(t, st.RW())
	ctx := t.Context()

	id1, err := createSession(ctx, st.RW(), agentID, modelID, "sys", nil, nil)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if err := appendTurn(ctx, st.RW(), id1, "prompt one", &goai.TextResult{Text: "reply one"}); err != nil {
		t.Fatalf("appendTurn: %v", err)
	}
	id2, err := createSession(ctx, st.RW(), agentID, modelID, "sys", nil, nil)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if err := appendTurn(ctx, st.RW(), id2, "prompt two", &goai.TextResult{Text: "reply two"}); err != nil {
		t.Fatalf("appendTurn: %v", err)
	}

	turns, err := SearchHistory(ctx, st, &id1, "", 10)
	if err != nil {
		t.Fatalf("SearchHistory: %v", err)
	}
	if len(turns) != 1 || turns[0].SessionID != id1 {
		t.Fatalf("turns = %+v, want only session %s", turns, id1)
	}
}

func TestDBSearchHistoryLimit(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	agentID, modelID := seedAgent(t, st.RW())
	ctx := t.Context()

	id, err := createSession(ctx, st.RW(), agentID, modelID, "sys", nil, nil)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if err := appendTurn(ctx, st.RW(), id, "turn one", &goai.TextResult{Text: "reply one"}); err != nil {
		t.Fatalf("appendTurn: %v", err)
	}
	if err := appendTurn(ctx, st.RW(), id, "turn two", &goai.TextResult{Text: "reply two"}); err != nil {
		t.Fatalf("appendTurn: %v", err)
	}

	turns, err := SearchHistory(ctx, st, &id, "", 1)
	if err != nil {
		t.Fatalf("SearchHistory: %v", err)
	}
	if len(turns) != 1 || turns[0].Prompt != "turn two" {
		t.Fatalf("turns = %+v, want only the newest turn", turns)
	}
}
