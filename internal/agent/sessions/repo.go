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

// AgentByName resolves a named agent's model and prompt template in one
// round trip. agent_name is expected to be unique per deployment.
func AgentByName(ctx context.Context, db *sql.DB, name string) (AgentConfig, error) {
	var (
		agentIDStr string
		modelIDStr string
		cfg        AgentConfig
	)
	err := db.QueryRowContext(ctx, `
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
func AgentIDFor(ctx context.Context, db *sql.DB, id SessionID) (AgentID, error) {
	var agentIDStr string
	err := db.QueryRowContext(ctx, `
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
func ToolNamesForAgent(ctx context.Context, db *sql.DB, agentID AgentID) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
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
		sysPrompt, optsJSON, time.Now().UTC().Format(time.RFC3339Nano))
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

	res, err := db.ExecContext(ctx, `
		INSERT INTO session_turns (turn_id, session_id, prompt, result, created_at)
			SELECT ?, session_id, ?, ?, ?
		FROM agent_sessions
		WHERE session_id = ? AND deleted_at IS NULL`,
		uuid.NewV7().String(), prompt, resultJSON, time.Now().UTC().Format(time.RFC3339Nano), id.String())
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
