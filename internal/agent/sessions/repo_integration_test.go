package sessions

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"

	"github.com/mhmdkzr/loop/internal/store"
	"github.com/mhmdkzr/loop/migrations"
	"github.com/mhmdkzr/loop/pkg/migrate"
	"github.com/mhmdkzr/loop/pkg/testenv"
)

// openTestStore opens a migrated Store backed by a fresh temp-file database.
func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "sessions.sqlite"))
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
	promptQuery := `INSERT INTO prompt_templates (prompt_id, prompt_name, template_body, params_schema, version) VALUES (?, ?, ?, ?, 1)`
	if _, err := db.ExecContext(ctx, promptQuery,
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

// TestDBReconcileInterruptedTurn simulates a crash mid-turn: a turn is
// started (the write-ahead marker) and given three step_finish tool calls,
// but only one ever gets both a tool_call_start and tool_call_result event,
// one gets only tool_call_start (started, outcome unknown), and one gets
// neither (never attempted) - mirroring a process that died partway through
// a parallel tool-call batch. ReconcileInterrupted must close the turn out
// with a synthesized result distinguishing all three outcomes, rather than
// leaving it stuck turnStatusRunning or losing the two real events.
func TestDBReconcileInterruptedTurn(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	agentID, modelID := seedAgent(t, st.RW())
	ctx := t.Context()

	sessionID, err := createSession(ctx, st.RW(), agentID, modelID, "sys", nil, nil)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}

	turnID := uuid.NewV7()
	if err := startTurn(ctx, st.RW(), sessionID, turnID, "run the deploy checks"); err != nil {
		t.Fatalf("startTurn: %v", err)
	}

	events := []struct {
		eventType string
		payload   any
	}{
		{"step_finish", stepFinishPayload{
			Step: 1,
			Text: "running checks",
			ToolCalls: []turnEventToolCall{
				{ID: "call-done", Name: "toolA"},
				{ID: "call-started", Name: "toolB"},
				{ID: "call-untouched", Name: "toolC"},
			},
		}},
		{
			"tool_call_start",
			toolCallStartPayload{Step: 1, ToolCallID: "call-done", ToolName: "toolA"},
		},
		{
			"tool_call_start",
			toolCallStartPayload{Step: 1, ToolCallID: "call-started", ToolName: "toolB"},
		},
		{
			"tool_call_result",
			toolCallResultPayload{
				Step:       1,
				ToolCallID: "call-done",
				ToolName:   "toolA",
				Output:     "all green",
			},
		},
	}
	for i, e := range events {
		if err := recordTurnEvent(ctx, st.RW(), turnID, int64(i+1), e.eventType, e.payload); err != nil {
			t.Fatalf("recordTurnEvent(%s): %v", e.eventType, err)
		}
	}

	reconciled, err := ReconcileInterrupted(ctx, st)
	if err != nil {
		t.Fatalf("ReconcileInterrupted: %v", err)
	}
	if reconciled != 1 {
		t.Fatalf("reconciled = %d, want 1", reconciled)
	}

	turns, err := turnsBySession(ctx, st.RO(), sessionID)
	if err != nil {
		t.Fatalf("turnsBySession: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("len(turns) = %d, want 1", len(turns))
	}
	got := turns[0]
	if got.Status != turnStatusInterrupted {
		t.Fatalf("status = %q, want %q", got.Status, turnStatusInterrupted)
	}
	if got.Result == nil {
		t.Fatalf("result = nil, want synthesized partial result")
	}

	toolOutputs := make(map[string]string)
	for _, msg := range got.Result.ResponseMessages {
		if msg.Role != provider.RoleTool {
			continue
		}
		for _, part := range msg.Content {
			if part.Type == provider.PartToolResult {
				toolOutputs[part.ToolCallID] = part.ToolOutput
			}
		}
	}

	if got := toolOutputs["call-done"]; got != "all green" {
		t.Errorf("call-done output = %q, want %q", got, "all green")
	}
	if got := toolOutputs["call-started"]; !strings.Contains(got, "outcome is unknown") {
		t.Errorf("call-started output = %q, want it to flag an unknown outcome", got)
	}
	if got := toolOutputs["call-untouched"]; !strings.Contains(got, "was ever attempted") {
		t.Errorf("call-untouched output = %q, want it to flag it was never attempted", got)
	}

	// A second reconciliation pass must be a no-op: the turn is no longer
	// turnStatusRunning, so it should neither be picked up again nor error.
	reconciled, err = ReconcileInterrupted(ctx, st)
	if err != nil {
		t.Fatalf("ReconcileInterrupted (second pass): %v", err)
	}
	if reconciled != 0 {
		t.Fatalf("second pass reconciled = %d, want 0", reconciled)
	}
}

func TestDBTokenUsage(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	agentID, modelID := seedAgent(t, st.RW())
	ctx := t.Context()

	id1, err := createSession(ctx, st.RW(), agentID, modelID, "sys", nil, nil)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	id2, err := createSession(ctx, st.RW(), agentID, modelID, "sys", nil, nil)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}

	usage1 := provider.Usage{
		InputTokens: 10, OutputTokens: 20, TotalTokens: 30,
		ReasoningTokens: 4, CacheReadTokens: 5, CacheWriteTokens: 6,
	}
	usage2 := provider.Usage{
		InputTokens: 2, OutputTokens: 3, TotalTokens: 5,
		ReasoningTokens: 1, CacheReadTokens: 7, CacheWriteTokens: 8,
	}
	if err := appendTurn(ctx, st.RW(), id1, "first", &goai.TextResult{TotalUsage: usage1}); err != nil {
		t.Fatalf("appendTurn: %v", err)
	}
	if err := appendTurn(ctx, st.RW(), id2, "second", &goai.TextResult{TotalUsage: usage2}); err != nil {
		t.Fatalf("appendTurn: %v", err)
	}

	got, err := GetSessionTokenUsage(ctx, st.RO(), id1.UUID())
	if err != nil {
		t.Fatalf("GetSessionTokenUsage: %v", err)
	}
	if got != usage1 {
		t.Fatalf("session usage = %+v, want %+v", got, usage1)
	}

	got, err = GetTotalTokenUsage(ctx, st.RO(), []uuid.UUID{id1.UUID(), id2.UUID()})
	if err != nil {
		t.Fatalf("GetTotalTokenUsage: %v", err)
	}
	want := provider.Usage{
		InputTokens: 12, OutputTokens: 23, TotalTokens: 35,
		ReasoningTokens: 5, CacheReadTokens: 12, CacheWriteTokens: 14,
	}
	if got != want {
		t.Fatalf("total usage = %+v, want %+v", got, want)
	}

	got, err = GetTotalTokenUsage(ctx, st.RO(), nil)
	if err != nil {
		t.Fatalf("GetTotalTokenUsage empty: %v", err)
	}
	if got != (provider.Usage{}) {
		t.Fatalf("empty total usage = %+v, want zero", got)
	}
}
