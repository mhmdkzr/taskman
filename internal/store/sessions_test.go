package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/mhmdkzr/taskman/internal/usage"
)

func TestSessionRoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	tools := []Tool{{Name: "bash", Description: "Run a shell command", InputSchema: `{"type":"object"}`}}
	sessionID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := CreateSession(ctx, db, sessionID, "deepseek-v4-flash", "medium", 100, "You are a helpful assistant.", tools); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	callID := "call_00_33y6gCqg4vzYxIjBgySJ4557"
	if err := AppendRun(ctx, db, sessionID, Run{
		Prompt: "what time is it?",
		Messages: []Message{
			{
				Role: "assistant", Number: 1, FinishReason: "tool-calls",
				Usage: usage.TokenUsage{InputTokens: 120, OutputTokens: 63, TotalTokens: 439, ReasoningTokens: 20, CacheReadTokens: 256},
				Parts: []Part{
					{Type: "reasoning", Text: "I can use a shell command to get the time."},
					{Type: "tool-call", ToolCallID: callID, ToolName: "bash", ToolInput: `{"command":"date"}`},
				},
			},
			{
				Role: "tool", Number: 1,
				Parts: []Part{
					{Type: "tool-result", ToolCallID: callID, ToolName: "bash", ToolOutput: `{"exit_code":0,"stdout":"Thu Aug 13 02:58:02 +0330 2026\n"}`},
				},
			},
			{
				Role: "assistant", Number: 2, FinishReason: "stop",
				Usage: usage.TokenUsage{InputTokens: 94, OutputTokens: 73, TotalTokens: 551, ReasoningTokens: 34, CacheReadTokens: 384},
				Parts: []Part{{Type: "text", Text: "The current time is 02:58 AM."}},
			},
		},
	}); err != nil {
		t.Fatalf("AppendRun: %v", err)
	}

	sess, err := GetSession(ctx, db, sessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess.Model != "deepseek-v4-flash" || sess.MaxSteps != 100 {
		t.Errorf("session = %+v, want model deepseek-v4-flash maxSteps 100", sess)
	}
	if sess.SystemPrompt != "You are a helpful assistant." {
		t.Errorf("system prompt = %q", sess.SystemPrompt)
	}
	if len(sess.Tools) != 1 || sess.Tools[0].Name != "bash" {
		t.Errorf("session tools = %+v, want [bash]", sess.Tools)
	}

	tr, err := GetSessionTranscript(ctx, db, sessionID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 1 {
		t.Fatalf("turns = %d, want 1", len(tr.Turns))
	}
	tt := tr.Turns[0]
	if tt.Prompt.Role != "user" || len(tt.Prompt.Parts) != 1 || tt.Prompt.Parts[0].Text != "what time is it?" {
		t.Errorf("turn prompt = %+v", tt.Prompt)
	}
	if len(tt.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(tt.Steps))
	}

	first := tt.Steps[0]
	if first.Step.FinishReason != "tool-calls" || first.Step.Usage.TotalTokens != 439 {
		t.Errorf("step1 = %+v", first.Step)
	}
	if len(first.Messages) != 2 {
		t.Fatalf("step1 messages = %d, want 2", len(first.Messages))
	}
	assistant := first.Messages[0]
	found := false
	for _, p := range assistant.Parts {
		if p.Type == "tool-call" && p.ToolCallID == callID && p.ToolName == "bash" && p.ToolInput == `{"command":"date"}` {
			found = true
		}
	}
	if !found {
		t.Errorf("assistant tool-call part not linked: %+v", assistant.Parts)
	}
	toolMsg := first.Messages[1]
	if toolMsg.Parts[0].Type != "tool-result" || toolMsg.Parts[0].ToolOutput == "" {
		t.Errorf("tool-result part missing output: %+v", toolMsg.Parts)
	}

	last := tt.Steps[1]
	if last.Step.FinishReason != "stop" || last.Step.Usage.CacheReadTokens != 384 {
		t.Errorf("step2 = %+v", last.Step)
	}
	if len(last.Messages) != 1 || last.Messages[0].Parts[0].Text != "The current time is 02:58 AM." {
		t.Errorf("final message = %+v", last.Messages)
	}

	sessions, err := ListSessions(ctx, db)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("listed sessions = %d, want 1", len(sessions))
	}

	if err := DeleteSession(ctx, db, sessionID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := GetSession(ctx, db, sessionID); err == nil {
		t.Error("GetSession after delete: want error")
	}
}

// TestNewSessionID verifies session ids are fresh, non-empty UUIDv7s whose
// embedded timestamp is recoverable (it becomes the session's created_at).
func TestNewSessionID(t *testing.T) {
	a, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	b, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if a == "" || b == "" {
		t.Fatal("empty session id")
	}
	if a == b {
		t.Fatal("NewSessionID returned the same id twice")
	}
	if _, err := uuid.Parse(a); err != nil {
		t.Errorf("session id %q is not a valid UUID: %v", a, err)
	}
	if ts := timestampFromID(a); ts == "" {
		t.Error("timestampFromID returned empty timestamp")
	}
}

// TestSaveRunCreatedAtMatchesID verifies a run's session row stores the
// pre-generated id and a created_at derived from the id's embedded timestamp.
func TestSaveRunCreatedAtMatchesID(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	sessionID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := SaveRun(ctx, db, sessionID, Run{Model: "m", Prompt: "hi"}); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	var createdAt string
	if err := db.QueryRowContext(ctx, `SELECT created_at FROM sessions WHERE id = ?`, sessionID).
		Scan(&createdAt); err != nil {
		t.Fatalf("load session: %v", err)
	}
	if want := timestampFromID(sessionID); createdAt != want {
		t.Errorf("created_at = %q, want %q (from the session id)", createdAt, want)
	}
}

func TestSaveRun(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	callID := "call_01"
	run := Run{
		Model:           "deepseek-v4-flash",
		ReasoningEffort: "medium",
		MaxSteps:        100,
		SystemPrompt:    "You are a helpful assistant.",
		Prompt:          "what time is it?",
		Tools:           []Tool{{Name: "bash", Description: "Run a shell command", InputSchema: `{"type":"object"}`}},
		Messages: []Message{
			{
				Role: "assistant", Number: 1, FinishReason: "tool-calls",
				Usage: usage.TokenUsage{InputTokens: 120, OutputTokens: 63, TotalTokens: 439, ReasoningTokens: 20, CacheReadTokens: 256},
				Parts: []Part{
					{Type: "reasoning", Text: "I can run a shell command."},
					{Type: "tool-call", ToolCallID: callID, ToolName: "bash", ToolInput: `{"command":"date"}`},
				},
			},
			{
				Role: "tool", Number: 1,
				Parts: []Part{{Type: "tool-result", ToolCallID: callID, ToolName: "bash", ToolOutput: `{"exit_code":0,"stdout":"Thu Aug 13 02:58:02 +0330 2026\n"}`}},
			},
			{
				Role: "assistant", Number: 2, FinishReason: "stop",
				Usage: usage.TokenUsage{InputTokens: 94, OutputTokens: 73, TotalTokens: 551, ReasoningTokens: 34, CacheReadTokens: 384},
				Parts: []Part{{Type: "text", Text: "The current time is 02:58 AM."}},
			},
		},
	}

	sessionID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := SaveRun(ctx, db, sessionID, run); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	tr, err := GetSessionTranscript(ctx, db, sessionID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if tr.Session.SystemPrompt != run.SystemPrompt || tr.Session.Model != run.Model {
		t.Errorf("session = %+v", tr.Session)
	}
	if len(tr.Turns) != 1 {
		t.Fatalf("turns = %d, want 1", len(tr.Turns))
	}
	tt := tr.Turns[0]
	if len(tt.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(tt.Steps))
	}
	if got := len(tt.Steps[0].Messages); got != 2 {
		t.Errorf("step1 messages = %d, want 2", got)
	}
	last := tt.Steps[1]
	if last.Step.FinishReason != "stop" || last.Messages[0].Parts[0].Text != "The current time is 02:58 AM." {
		t.Errorf("final step = %+v", last)
	}
}

// TestUsageColumnsNullable verifies the DB-level accounting convention: the
// usage columns are NULL on every message except the assistant message of a
// step, so "no usage recorded" is distinguishable from "used zero tokens".
func TestUsageColumnsNullable(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	callID := "call_01"
	run := Run{
		Model:        "m",
		SystemPrompt: "You are helpful.",
		Prompt:       "what time is it?",
		Messages: []Message{
			{Role: "assistant", Number: 1, FinishReason: "tool-calls",
				Usage: usage.TokenUsage{InputTokens: 120, OutputTokens: 63, TotalTokens: 439},
				Parts: []Part{
					{Type: "tool-call", ToolCallID: callID, ToolName: "bash", ToolInput: `{"command":"date"}`},
				}},
			{Role: "tool", Number: 1,
				Parts: []Part{{Type: "tool-result", ToolCallID: callID, ToolName: "bash", ToolOutput: "now"}}},
			{Role: "assistant", Number: 2, FinishReason: "stop",
				Usage: usage.TokenUsage{InputTokens: 94, OutputTokens: 73, TotalTokens: 551},
				Parts: []Part{{Type: "text", Text: "done"}}},
		},
	}

	sessionID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := SaveRun(ctx, db, sessionID, run); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	// This test's DB holds a single session, so all message rows are its path.
	rows, err := db.QueryContext(ctx, `
		SELECT role, number, input_tokens, output_tokens, total_tokens
		FROM messages`)
	if err != nil {
		t.Fatalf("query messages: %v", err)
	}
	defer rows.Close()

	type gotMsg struct {
		role            string
		number          sql.NullInt64
		inTok, outTotok sql.NullInt64
		total           sql.NullInt64
	}
	var got []gotMsg
	for rows.Next() {
		var g gotMsg
		if err := rows.Scan(&g.role, &g.number, &g.inTok, &g.outTotok, &g.total); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, g)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	// system, user, tool: all usage NULL. assistant: populated.
	for _, g := range got {
		switch g.role {
		case "system", "user", "tool":
			if g.inTok.Valid || g.outTotok.Valid || g.total.Valid {
				t.Errorf("%s message usage = %+v, want NULL columns", g.role, g)
			}
		case "assistant":
			if !g.inTok.Valid || !g.total.Valid || g.total.Int64 != 439 && g.total.Int64 != 551 {
				t.Errorf("assistant usage = %+v, want populated", g)
			}
		}
	}
}

// TestSaveRunConcurrent spawns many writers against one file DB, each saving a
// distinct run. Every run must be fully present afterwards (transactional) and
// all runs must be stored (no lost updates from racing writers).
func TestSaveRunConcurrent(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "taskman.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const n = 16
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx := context.Background()
			run := Run{
				Model:           "deepseek-v4-flash",
				ReasoningEffort: "medium",
				MaxSteps:        100,
				SystemPrompt:    "shared system prompt",
				Prompt:          fmt.Sprintf("prompt %d", i),
				Tools:           []Tool{{Name: "bash", Description: "Run a shell command"}},
				Messages: []Message{{
					Role: "assistant", Number: 1, FinishReason: "stop",
					Usage: usage.TokenUsage{InputTokens: i + 1, OutputTokens: 1, TotalTokens: i + 2},
					Parts: []Part{{Type: "text", Text: fmt.Sprintf("answer %d", i)}},
				}},
			}
			sessionID, err := NewSessionID()
			if err != nil {
				errs[i] = err
				return
			}
			if err := SaveRun(ctx, db, sessionID, run); err != nil {
				errs[i] = err
			}
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("writer %d: %v", i, err)
		}
	}

	sessions, err := ListSessions(context.Background(), db)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != n {
		t.Fatalf("sessions = %d, want %d", len(sessions), n)
	}

	prompts := map[string]bool{}
	for _, s := range sessions {
		tr, err := GetSessionTranscript(context.Background(), db, s.ID)
		if err != nil {
			t.Fatalf("transcript %s: %v", s.ID, err)
		}
		if len(tr.Turns) != 1 || len(tr.Turns[0].Steps) != 1 {
			t.Fatalf("session %s partial: turns=%d steps=%d", s.ID, len(tr.Turns), len(tr.Turns[0].Steps))
		}
		prompts[tr.Turns[0].Prompt.Parts[0].Text] = true
	}
	if len(prompts) != n {
		t.Errorf("distinct prompts stored = %d, want %d", len(prompts), n)
	}
}

func TestAppendRun(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	sessionID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := SaveRun(ctx, db, sessionID, Run{
		Model:        "deepseek-v4-flash",
		Prompt:       "first turn",
		SystemPrompt: "You are helpful.",
		Messages: []Message{{
			Role: "assistant", Number: 1, FinishReason: "stop",
			Usage: usage.TokenUsage{TotalTokens: 100},
			Parts: []Part{{Type: "text", Text: "first answer"}},
		}},
	}); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	if err := AppendRun(ctx, db, sessionID, Run{
		Prompt: "second turn",
		Messages: []Message{{
			Role: "assistant", Number: 1, FinishReason: "stop",
			Usage: usage.TokenUsage{TotalTokens: 200},
			Parts: []Part{{Type: "text", Text: "second answer"}},
		}},
	}); err != nil {
		t.Fatalf("AppendRun: %v", err)
	}

	tr, err := GetSessionTranscript(ctx, db, sessionID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 2 {
		t.Fatalf("turns = %d, want 2", len(tr.Turns))
	}
	if tr.Turns[0].Turn.Number != 1 || tr.Turns[0].Prompt.Parts[0].Text != "first turn" {
		t.Errorf("turn 1 = %+v", tr.Turns[0])
	}
	if tr.Turns[1].Turn.Number != 2 || tr.Turns[1].Prompt.Parts[0].Text != "second turn" {
		t.Errorf("turn 2 = %+v", tr.Turns[1])
	}
	if tr.Turns[1].Steps[0].Step.Usage.TotalTokens != 200 {
		t.Errorf("turn 2 step usage = %+v", tr.Turns[1].Steps[0].Step.Usage)
	}
}

func TestForkSession(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	run := func(prompt, answer string) Run {
		return Run{
			Model:        "deepseek-v4-flash",
			SystemPrompt: "You are helpful.",
			Prompt:       prompt,
			Tools:        []Tool{{Name: "bash", Description: "Run a shell command"}},
			Messages: []Message{{
				Role: "assistant", Number: 1, FinishReason: "stop",
				Usage: usage.TokenUsage{TotalTokens: 100},
				Parts: []Part{{Type: "text", Text: answer}},
			}},
		}
	}

	srcID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := SaveRun(ctx, db, srcID, run("first prompt", "first answer")); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}
	if err := AppendRun(ctx, db, srcID, run("second prompt", "second answer")); err != nil {
		t.Fatalf("AppendRun: %v", err)
	}

	srcTr, err := GetSessionTranscript(ctx, db, srcID)
	if err != nil {
		t.Fatalf("GetSessionTranscript source: %v", err)
	}
	if len(srcTr.Turns) != 2 {
		t.Fatalf("source turns = %d, want 2", len(srcTr.Turns))
	}
	firstTurnID := srcTr.Turns[0].Turn.ID

	forkID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := ForkSession(ctx, db, srcID, forkID, 2); err != nil {
		t.Fatalf("ForkSession: %v", err)
	}
	if err := AppendRun(ctx, db, forkID, run("edited prompt", "edited answer")); err != nil {
		t.Fatalf("AppendRun fork: %v", err)
	}

	forkTr, err := GetSessionTranscript(ctx, db, forkID)
	if err != nil {
		t.Fatalf("GetSessionTranscript fork: %v", err)
	}
	if len(forkTr.Turns) != 2 {
		t.Fatalf("fork turns = %d, want 2", len(forkTr.Turns))
	}
	// By reference: the fork shares turn 1 (same turn/message ids), no copy.
	if forkTr.Turns[0].Turn.ID != firstTurnID {
		t.Error("fork turn 1 does not share the source turn id; want a shared reference")
	}
	if forkTr.Turns[0].Prompt.Parts[0].Text != "first prompt" {
		t.Errorf("fork turn 1 prompt = %q", forkTr.Turns[0].Prompt.Parts[0].Text)
	}
	if forkTr.Turns[1].Prompt.Parts[0].Text != "edited prompt" {
		t.Errorf("fork turn 2 prompt = %q, want edited prompt", forkTr.Turns[1].Prompt.Parts[0].Text)
	}
	if forkTr.Turns[1].Turn.Number != 2 {
		t.Errorf("fork turn 2 number = %d, want 2", forkTr.Turns[1].Turn.Number)
	}

	badID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := ForkSession(ctx, db, srcID, badID, 3); err == nil {
		t.Error("ForkSession at out-of-range turn: want error")
	}

	// Deleting the source leaves the fork intact: the shared messages survive
	// because the fork still references them.
	if err := DeleteSession(ctx, db, srcID); err != nil {
		t.Fatalf("DeleteSession source: %v", err)
	}
	forkTr, err = GetSessionTranscript(ctx, db, forkID)
	if err != nil {
		t.Fatalf("transcript after source delete: %v", err)
	}
	if len(forkTr.Turns) != 2 {
		t.Fatalf("fork turns after source delete = %d, want 2", len(forkTr.Turns))
	}
}

// TestForkLineage verifies a fork records its parent and fork point in the
// forks table.
func TestForkLineage(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	srcID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := SaveRun(ctx, db, srcID, Run{Model: "m", Prompt: "first"}); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}
	if err := AppendRun(ctx, db, srcID, Run{Prompt: "second"}); err != nil {
		t.Fatalf("AppendRun: %v", err)
	}

	forkID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := ForkSession(ctx, db, srcID, forkID, 2); err != nil {
		t.Fatalf("ForkSession: %v", err)
	}

	var parentID string
	var turnNumber int
	if err := db.QueryRowContext(ctx,
		`SELECT parent_id, turn_number FROM forks WHERE session_id = ?`, forkID).
		Scan(&parentID, &turnNumber); err != nil {
		t.Fatalf("load fork lineage: %v", err)
	}
	if parentID != srcID || turnNumber != 2 {
		t.Errorf("lineage = (%s, %d), want (%s, 2)", parentID, turnNumber, srcID)
	}

	sess, err := GetSession(ctx, db, forkID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess.ParentID != srcID || sess.ForkTurnNumber != 2 {
		t.Errorf("session lineage = (%q, %d)", sess.ParentID, sess.ForkTurnNumber)
	}

	// A session with no parent has empty lineage.
	plain, err := GetSession(ctx, db, srcID)
	if err != nil {
		t.Fatalf("GetSession source: %v", err)
	}
	if plain.ParentID != "" || plain.ForkTurnNumber != 0 {
		t.Errorf("source lineage = (%q, %d), want empty", plain.ParentID, plain.ForkTurnNumber)
	}

	// A fork_turn_number beyond the source's turns is rejected at fork time.
	badID, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := ForkSession(ctx, db, srcID, badID, 99); err == nil {
		t.Error("ForkSession at turn 99: want error")
	}
}

// TestForkOfFork verifies a fork of a fork sees the full effective transcript:
// the grandparent's shared turns, the parent's diverged turns, and its own. The
// fork references the parent's leaf, so no message rows are ever copied.
func TestForkOfFork(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	run := func(prompt, answer string) Run {
		return Run{
			Model:        "deepseek-v4-flash",
			SystemPrompt: "You are helpful.",
			Prompt:       prompt,
			Tools:        []Tool{{Name: "bash", Description: "Run a shell command"}},
			Messages: []Message{{
				Role: "assistant", Number: 1, FinishReason: "stop",
				Usage: usage.TokenUsage{TotalTokens: 100},
				Parts: []Part{{Type: "text", Text: answer}},
			}},
		}
	}
	mustSave := func(sessionID, prompt, answer string) {
		t.Helper()
		if err := SaveRun(ctx, db, sessionID, run(prompt, answer)); err != nil {
			t.Fatalf("SaveRun %q: %v", prompt, err)
		}
	}
	mustAppend := func(sessionID, prompt, answer string) {
		t.Helper()
		if err := AppendRun(ctx, db, sessionID, run(prompt, answer)); err != nil {
			t.Fatalf("AppendRun %q: %v", prompt, err)
		}
	}

	srcID, _ := NewSessionID()
	mustSave(srcID, "first prompt", "first answer")
	mustAppend(srcID, "second prompt", "second answer")
	mustAppend(srcID, "third prompt", "third answer")

	forkID, _ := NewSessionID()
	if err := ForkSession(ctx, db, srcID, forkID, 2); err != nil {
		t.Fatalf("ForkSession: %v", err)
	}
	mustAppend(forkID, "forked second", "forked second answer")
	mustAppend(forkID, "forked third", "forked third answer")

	// Fork the fork at its turn 3, replacing "forked third".
	deepID, _ := NewSessionID()
	if err := ForkSession(ctx, db, forkID, deepID, 3); err != nil {
		t.Fatalf("ForkSession deep: %v", err)
	}
	mustAppend(deepID, "deep fork", "deep answer")

	tr, err := GetSessionTranscript(ctx, db, deepID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 3 {
		t.Fatalf("deep fork turns = %d, want 3", len(tr.Turns))
	}
	want := []string{"first prompt", "forked second", "deep fork"}
	for i, tt := range tr.Turns {
		if tt.Prompt.Parts[0].Text != want[i] {
			t.Errorf("deep fork turn %d prompt = %q, want %q", i+1, tt.Prompt.Parts[0].Text, want[i])
		}
		if tt.Turn.Number != i+1 {
			t.Errorf("deep fork turn %d number = %d, want %d", i+1, tt.Turn.Number, i+1)
		}
	}
	if tr.Session.SystemPrompt != "You are helpful." {
		t.Errorf("deep fork system prompt = %q", tr.Session.SystemPrompt)
	}

	sess, err := GetSession(ctx, db, deepID)
	if err != nil {
		t.Fatalf("GetSession deep: %v", err)
	}
	if sess.ParentID != forkID || sess.ForkTurnNumber != 3 {
		t.Errorf("deep lineage = (%q, %d)", sess.ParentID, sess.ForkTurnNumber)
	}
}

// TestLineageSurvivesParentDeletion verifies deleting a fork's parent clears
// only the parent reference (ON DELETE SET NULL); the fork's transcript and
// fork point survive.
func TestLineageSurvivesParentDeletion(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	srcID, _ := NewSessionID()
	if err := SaveRun(ctx, db, srcID, Run{
		Model:        "m",
		SystemPrompt: "You are helpful.",
		Prompt:       "first",
		Messages: []Message{{Role: "assistant", Number: 1, FinishReason: "stop",
			Usage: usage.TokenUsage{TotalTokens: 10}, Parts: []Part{{Type: "text", Text: "a"}}}},
	}); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}
	if err := AppendRun(ctx, db, srcID, Run{Prompt: "second"}); err != nil {
		t.Fatalf("AppendRun: %v", err)
	}

	forkID, _ := NewSessionID()
	if err := ForkSession(ctx, db, srcID, forkID, 2); err != nil {
		t.Fatalf("ForkSession: %v", err)
	}
	if err := AppendRun(ctx, db, forkID, Run{Prompt: "forked second"}); err != nil {
		t.Fatalf("AppendRun fork: %v", err)
	}

	if err := DeleteSession(ctx, db, srcID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	sess, err := GetSession(ctx, db, forkID)
	if err != nil {
		t.Fatalf("GetSession fork: %v", err)
	}
	if sess.ParentID != "" {
		t.Errorf("ParentID = %q, want empty after parent deletion", sess.ParentID)
	}
	if sess.ForkTurnNumber != 2 {
		t.Errorf("ForkTurnNumber = %d, want 2 preserved", sess.ForkTurnNumber)
	}

	var parent sql.NullString
	var turnNumber int
	if err := db.QueryRowContext(ctx,
		`SELECT parent_id, turn_number FROM forks WHERE session_id = ?`, forkID).
		Scan(&parent, &turnNumber); err != nil {
		t.Fatalf("load fork lineage: %v", err)
	}
	if parent.Valid {
		t.Errorf("forks.parent_id = %v, want NULL", parent)
	}
	if turnNumber != 2 {
		t.Errorf("forks.turn_number = %d, want 2", turnNumber)
	}

	tr, err := GetSessionTranscript(ctx, db, forkID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 2 {
		t.Fatalf("fork turns after parent delete = %d, want 2", len(tr.Turns))
	}
	if tr.Session.SystemPrompt != "You are helpful." {
		t.Errorf("fork system prompt after parent delete = %q", tr.Session.SystemPrompt)
	}
}

// TestForkAtTurnOne verifies a fork at turn 1 shares no turns (only the system
// message when present) and its first run is turn 1.
func TestForkAtTurnOne(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	srcID, _ := NewSessionID()
	if err := SaveRun(ctx, db, srcID, Run{
		Model:        "m",
		SystemPrompt: "You are helpful.",
		Prompt:       "first",
		Messages: []Message{{Role: "assistant", Number: 1, FinishReason: "stop",
			Usage: usage.TokenUsage{TotalTokens: 10}, Parts: []Part{{Type: "text", Text: "a"}}}},
	}); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	forkID, _ := NewSessionID()
	if err := ForkSession(ctx, db, srcID, forkID, 1); err != nil {
		t.Fatalf("ForkSession: %v", err)
	}

	tr, err := GetSessionTranscript(ctx, db, forkID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 0 {
		t.Errorf("fork at turn 1 shares no turns; got %d", len(tr.Turns))
	}
	if tr.Session.SystemPrompt != "You are helpful." {
		t.Errorf("fork at turn 1 system prompt = %q", tr.Session.SystemPrompt)
	}

	if err := AppendRun(ctx, db, forkID, Run{Prompt: "first of fork"}); err != nil {
		t.Fatalf("AppendRun fork: %v", err)
	}
	tr, err = GetSessionTranscript(ctx, db, forkID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 1 || tr.Turns[0].Turn.Number != 1 {
		t.Fatalf("fork turn 1 = %+v, want a single turn numbered 1", tr.Turns)
	}
	if tr.Turns[0].Prompt.Parts[0].Text != "first of fork" {
		t.Errorf("fork turn 1 prompt = %q", tr.Turns[0].Prompt.Parts[0].Text)
	}
}

// TestChainRestrictsMidChainDeletion verifies the ON DELETE RESTRICT guards: a
// message that is an interior chain link or a session's leaf cannot be deleted,
// so a transcript can never be silently severed.
func TestChainRestrictsMidChainDeletion(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	sessionID, _ := NewSessionID()
	if err := SaveRun(ctx, db, sessionID, Run{
		Model:        "m",
		SystemPrompt: "You are helpful.",
		Prompt:       "what time is it?",
		Messages: []Message{
			{Role: "assistant", Number: 1, FinishReason: "stop",
				Usage: usage.TokenUsage{TotalTokens: 10}, Parts: []Part{{Type: "text", Text: "answer"}}},
		},
	}); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}

	sess, err := GetSession(ctx, db, sessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	path, err := chainPath(ctx, db, sess.LeafMessageID)
	if err != nil {
		t.Fatalf("chainPath: %v", err)
	}
	if len(path) != 3 { // system, user, assistant
		t.Fatalf("path = %d messages, want 3", len(path))
	}

	mid := path[1] // the user prompt, an interior chain link
	if _, err := db.ExecContext(ctx, `DELETE FROM messages WHERE id = ?`, mid.ID); err == nil {
		t.Error("deleting an interior message: want RESTRICT error")
	}
	leaf := path[len(path)-1] // the session's leaf
	if _, err := db.ExecContext(ctx, `DELETE FROM messages WHERE id = ?`, leaf.ID); err == nil {
		t.Error("deleting a session's leaf: want RESTRICT error")
	}

	tr, err := GetSessionTranscript(ctx, db, sessionID)
	if err != nil {
		t.Fatalf("GetSessionTranscript: %v", err)
	}
	if len(tr.Turns) != 1 || len(tr.Turns[0].Steps) != 1 {
		t.Fatalf("transcript after blocked deletes = %+v", tr.Turns)
	}
}
