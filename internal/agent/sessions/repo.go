package sessions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"uuid"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"

	"github.com/mhmdkzr/loop/internal/store"
)

var (
	// errSessionNotFound is returned when no live session matches the requested ID.
	errSessionNotFound = errors.New("session not found")

	// errAgentNotFound is returned when no agents row matches the requested name.
	errAgentNotFound = errors.New("agent not found")
)

// sessionConfig is the immutable configuration of an agent session plus its
// turn history loaded on demand.
type sessionConfig struct {
	ModelID         ModelID
	SystemPrompt    string
	ProviderOptions map[string]any
	Turns           []turn
}

// turn is one user prompt and the full goai.TextResult it produced.
type turn struct {
	Prompt string
	Result *goai.TextResult
}

// AgentConfig is a named agent's session-relevant definition: the model it
// runs on and the prompt template its system prompt is rendered from.
type AgentConfig struct {
	AgentID      AgentID
	ModelID      ModelID
	TemplateBody string
}

// HistoryTurn is one session_turns row joined with its parent session: the
// user's prompt and the assistant's final reply text.
type HistoryTurn struct {
	SessionID        SessionID
	SessionCreatedAt string
	Prompt           string
	Reply            string
	CreatedAt        string
}

// AgentByName resolves a named agent's model and prompt template in one
// round trip. agent_name is expected to be unique per deployment.
func AgentByName(ctx context.Context, st *store.Store, name string) (AgentConfig, error) {
	var (
		agentIDStr string
		modelIDStr string
		cfg        AgentConfig
	)
	err := st.RO().QueryRowContext(ctx, `
		SELECT a.agent_id, a.model_id, pt.template_body
		FROM agents a
		JOIN prompt_templates pt ON pt.prompt_id = a.prompt_id
		WHERE a.agent_name = ?`, name).
		Scan(&agentIDStr, &modelIDStr, &cfg.TemplateBody)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentConfig{}, errAgentNotFound
	}
	if err != nil {
		return AgentConfig{}, fmt.Errorf("query agent by name: %w", err)
	}

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		return AgentConfig{}, fmt.Errorf("parse agent id: %w", err)
	}
	modelID, err := uuid.Parse(modelIDStr)
	if err != nil {
		return AgentConfig{}, fmt.Errorf("parse model id: %w", err)
	}
	cfg.AgentID = AgentID(agentID)
	cfg.ModelID = ModelID(modelID)
	return cfg, nil
}

// AgentIDFor resolves a live session to the agent it runs as.
func AgentIDFor(ctx context.Context, st *store.Store, id SessionID) (AgentID, error) {
	var agentIDStr string
	err := st.RO().QueryRowContext(ctx, `
		SELECT agent_id FROM agent_sessions
		WHERE session_id = ? AND deleted_at IS NULL`, id.String()).Scan(&agentIDStr)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentID{}, errSessionNotFound
	}
	if err != nil {
		return AgentID{}, fmt.Errorf("query agent id for session: %w", err)
	}
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		return AgentID{}, fmt.Errorf("parse agent id: %w", err)
	}
	return AgentID(agentID), nil
}

// ToolNamesForAgent returns the tool_name of every tool linked to agentID
// via agent_tools. An agent with no linked tools returns an empty slice.
func ToolNamesForAgent(ctx context.Context, st *store.Store, agentID AgentID) ([]string, error) {
	rows, err := st.RO().QueryContext(ctx, `
		SELECT t.tool_name
		FROM agent_tools at
		JOIN tools t ON t.tool_id = at.tool_id
		WHERE at.agent_id = ?
		ORDER BY t.tool_name`, agentID.String())
	if err != nil {
		return nil, fmt.Errorf("query tool names for agent: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close agent tool rows", "error", err)
		}
	}()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan tool name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tool names: %w", err)
	}
	return names, nil
}

func GetSessionTokenUsage(ctx context.Context, db *sql.DB, sessionID uuid.UUID) (provider.Usage, error) {
	var usage provider.Usage
	err := db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(total_tokens), 0),
			COALESCE(SUM(reasoning_tokens), 0),
			COALESCE(SUM(cache_read_tokens), 0),
			COALESCE(SUM(cache_write_tokens), 0)
		FROM token_usage
		WHERE session_id = ?`, sessionID.String()).Scan(
		&usage.InputTokens,
		&usage.OutputTokens,
		&usage.TotalTokens,
		&usage.ReasoningTokens,
		&usage.CacheReadTokens,
		&usage.CacheWriteTokens,
	)
	if err != nil {
		return provider.Usage{}, fmt.Errorf("query session token usage: %w", err)
	}
	return usage, nil
}

func GetTotalTokenUsage(ctx context.Context, db *sql.DB, sessionIDs []uuid.UUID) (provider.Usage, error) {
	if len(sessionIDs) == 0 {
		return provider.Usage{}, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(sessionIDs)), ",")
	args := make([]any, len(sessionIDs))
	for i, sessionID := range sessionIDs {
		args[i] = sessionID.String()
	}

	var usage provider.Usage
	err := db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(total_tokens), 0),
			COALESCE(SUM(reasoning_tokens), 0),
			COALESCE(SUM(cache_read_tokens), 0),
			COALESCE(SUM(cache_write_tokens), 0)
		FROM token_usage
		WHERE session_id IN (%s)`, placeholders), args...).Scan(
		&usage.InputTokens,
		&usage.OutputTokens,
		&usage.TotalTokens,
		&usage.ReasoningTokens,
		&usage.CacheReadTokens,
		&usage.CacheWriteTokens,
	)
	if err != nil {
		return provider.Usage{}, fmt.Errorf("query total token usage: %w", err)
	}
	return usage, nil
}

func recordSessionTokenUsage(
	ctx context.Context,
	tx *sql.Tx,
	sessionID uuid.UUID,
	turnID uuid.UUID,
	usage provider.Usage,
) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO token_usage (
			session_id, turn_id,
			input_tokens, output_tokens, total_tokens,
			reasoning_tokens, cache_read_tokens, cache_write_tokens,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID.String(), turnID.String(),
		usage.InputTokens, usage.OutputTokens, usage.TotalTokens,
		usage.ReasoningTokens, usage.CacheReadTokens, usage.CacheWriteTokens,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert token usage: %w", err)
	}
	return nil
}

// createSession validates and persists a new session with an empty turn
// history. parentSessionID is nil for a top-level, user-initiated session.
func createSession(
	ctx context.Context,
	db *sql.DB,
	agentID AgentID,
	modelID ModelID,
	sysPrompt string,
	providerOpts map[string]any,
	parentSessionID *SessionID,
) (SessionID, error) {
	if agentID == (AgentID{}) {
		return SessionID{}, fmt.Errorf("create session: agent id is required")
	}
	if modelID == (ModelID{}) {
		return SessionID{}, fmt.Errorf("create session: model id is required")
	}
	if strings.TrimSpace(sysPrompt) == "" {
		return SessionID{}, fmt.Errorf("create session: system prompt is required")
	}

	id := SessionID(uuid.NewV7())
	optsJSON, err := json.Marshal(providerOpts)
	if err != nil {
		return SessionID{}, fmt.Errorf("marshal session provider options: %w", err)
	}

	var parentIDVal any
	if parentSessionID != nil {
		if *parentSessionID == (SessionID{}) {
			return SessionID{}, fmt.Errorf("create session: parent session id is zero value")
		}
		parentIDVal = parentSessionID.String()
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO agent_sessions (
			session_id, agent_id, parent_session_id, model_id,
			system_prompt, provider_options, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id.String(), agentID.String(), parentIDVal, modelID.String(),
		sysPrompt, string(optsJSON), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return SessionID{}, fmt.Errorf("insert session: %w", err)
	}
	return id, nil
}

// sessionByID loads a live session's configuration and its full turn history.
func sessionByID(ctx context.Context, db *sql.DB, id SessionID) (sessionConfig, error) {
	const query = `
		SELECT model_id, system_prompt, provider_options
		FROM agent_sessions
		WHERE session_id = ? AND deleted_at IS NULL`

	var (
		cfg        sessionConfig
		modelIDStr string
		optsJSON   []byte
	)
	err := db.QueryRowContext(ctx, query, id.String()).
		Scan(&modelIDStr, &cfg.SystemPrompt, &optsJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return sessionConfig{}, errSessionNotFound
	}
	if err != nil {
		return sessionConfig{}, fmt.Errorf("query session by id: %w", err)
	}
	modelID, err := uuid.Parse(modelIDStr)
	if err != nil {
		return sessionConfig{}, fmt.Errorf("parse model id: %w", err)
	}
	cfg.ModelID = ModelID(modelID)
	if len(optsJSON) > 0 {
		if err := json.Unmarshal(optsJSON, &cfg.ProviderOptions); err != nil {
			return sessionConfig{}, fmt.Errorf("decode session provider options: %w", err)
		}
	}

	turns, err := turnsBySession(ctx, db, id)
	if err != nil {
		return sessionConfig{}, err
	}
	cfg.Turns = turns
	return cfg, nil
}

// modelNameByID resolves a models row identity to its model_name.
func modelNameByID(ctx context.Context, db *sql.DB, id ModelID) (string, error) {
	var name string
	err := db.QueryRowContext(ctx,
		`SELECT model_name FROM models WHERE model_id = ?`, id.String()).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("model not found")
	}
	if err != nil {
		return "", fmt.Errorf("query model by id: %w", err)
	}
	return name, nil
}

// appendTurn validates and records one user prompt and its full TextResult as a
// new turn. It inserts a row selected against the live session, so a missing or
// soft-deleted session inserts nothing and maps to errSessionNotFound.
func appendTurn(ctx context.Context, db *sql.DB, id SessionID, prompt string, result *goai.TextResult) error {
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("append turn: prompt is required")
	}
	if result == nil {
		return fmt.Errorf("append turn: result is required")
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal turn result: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin append turn transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			slog.Error("rollback append turn transaction", "error", rollbackErr)
		}
	}()

	turnID := uuid.NewV7()
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := tx.ExecContext(ctx, `
		INSERT INTO session_turns (turn_id, session_id, prompt, result, created_at)
			SELECT ?, session_id, ?, ?, ?
		FROM agent_sessions
		WHERE session_id = ? AND deleted_at IS NULL`,
		turnID.String(), prompt, string(resultJSON), createdAt, id.String())
	if err != nil {
		return fmt.Errorf("insert session turn: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return errSessionNotFound
	}
	if err := recordSessionTokenUsage(ctx, tx, uuid.UUID(id), turnID, result.TotalUsage); err != nil {
		return fmt.Errorf("record token usage: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit append turn transaction: %w", err)
	}
	return nil
}

// turnsBySession loads a session's turns in insertion order.
func turnsBySession(ctx context.Context, db *sql.DB, id SessionID) ([]turn, error) {
	const query = `
		SELECT prompt, result, created_at
		FROM session_turns
		WHERE session_id = ?
		ORDER BY created_at, turn_id`

	rows, err := db.QueryContext(ctx, query, id.String())
	if err != nil {
		return nil, fmt.Errorf("list session turns: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close session turn rows", "error", err)
		}
	}()

	var turns []turn
	for rows.Next() {
		var (
			t         turn
			resultRaw []byte
			createdAt string
		)
		if err := rows.Scan(&t.Prompt, &resultRaw, &createdAt); err != nil {
			return nil, fmt.Errorf("scan session turn: %w", err)
		}
		if len(resultRaw) > 0 {
			if err := json.Unmarshal(resultRaw, &t.Result); err != nil {
				return nil, fmt.Errorf("decode session turn result: %w", err)
			}
		}
		turns = append(turns, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session turns: %w", err)
	}
	return turns, nil
}

// SearchHistory returns turns across sessions, newest session first and
// newest turn first within a session. sessionID restricts the search to one
// session when non-nil; query matches turns whose prompt or final reply text
// contains it (case-insensitive substring); limit bounds the number of turns
// returned.
func SearchHistory(
	ctx context.Context,
	st *store.Store,
	sessionID *SessionID,
	query string,
	limit int,
) ([]HistoryTurn, error) {
	if st == nil {
		return nil, fmt.Errorf("search history: database is required")
	}
	if limit < 1 {
		return nil, fmt.Errorf("search history: limit must be positive")
	}
	return searchHistoryTurns(ctx, st.RO(), sessionID, query, limit)
}

// searchHistoryTurns streams session_turns newest first, decoding each
// result and stopping once limit matching turns are collected so a broad
// search does not require loading the whole table.
func searchHistoryTurns(
	ctx context.Context,
	db *sql.DB,
	sessionID *SessionID,
	query string,
	limit int,
) ([]HistoryTurn, error) {
	sessionFilter := ""
	if sessionID != nil {
		sessionFilter = sessionID.String()
	}

	rows, err := db.QueryContext(ctx, `
		SELECT s.session_id, s.created_at, t.prompt, t.result, t.created_at
		FROM session_turns t
		JOIN agent_sessions s ON s.session_id = t.session_id
		WHERE s.deleted_at IS NULL
			AND (? = '' OR s.session_id = ?)
		ORDER BY s.created_at DESC, t.created_at DESC, t.turn_id DESC`,
		sessionFilter, sessionFilter)
	if err != nil {
		return nil, fmt.Errorf("query session turns: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close history rows", "error", err)
		}
	}()

	query = strings.ToLower(query)
	var turns []HistoryTurn
	for rows.Next() {
		var (
			sessionIDStr     string
			sessionCreatedAt string
			prompt           string
			resultRaw        []byte
			createdAt        string
		)
		if err := rows.Scan(&sessionIDStr, &sessionCreatedAt, &prompt, &resultRaw, &createdAt); err != nil {
			return nil, fmt.Errorf("scan history turn: %w", err)
		}

		var result goai.TextResult
		if len(resultRaw) > 0 {
			if err := json.Unmarshal(resultRaw, &result); err != nil {
				return nil, fmt.Errorf("decode turn result: %w", err)
			}
		}

		if query != "" &&
			!strings.Contains(strings.ToLower(prompt), query) &&
			!strings.Contains(strings.ToLower(result.Text), query) {
			continue
		}

		id, err := uuid.Parse(sessionIDStr)
		if err != nil {
			return nil, fmt.Errorf("parse session id: %w", err)
		}
		turns = append(turns, HistoryTurn{
			SessionID:        SessionID(id),
			SessionCreatedAt: sessionCreatedAt,
			Prompt:           prompt,
			Reply:            result.Text,
			CreatedAt:        createdAt,
		})
		if len(turns) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history turns: %w", err)
	}
	return turns, nil
}
