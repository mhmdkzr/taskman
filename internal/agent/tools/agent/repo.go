package agent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"uuid"
)

var (
	ErrAgentNotFound = errors.New("agent not found")
	ErrToolNotFound  = errors.New("tool not found")
)

// CreateAgent persists a new agent definition: a's ID must be pre-generated
// by the caller, and CreateAgent generates the backing prompt_templates row
// (and its ID) as an implementation detail of the agent.
func CreateAgent(ctx context.Context, db *sql.DB, a Agent) error {
	if a.ID == (uuid.UUID{}) {
		return fmt.Errorf("create agent: id is required")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create agent: %w", err)
	}
	defer rollbackAgentTx(tx)

	promptID := uuid.NewV7()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO prompt_templates (prompt_id, prompt_name, template_body, params_schema, version)
		VALUES (?, ?, ?, ?, ?)`,
		promptID.String(), a.Name, a.TemplateBody, a.ParamsSchema, versionOrDefault(a.Version)); err != nil {
		return fmt.Errorf("insert prompt template: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO agents (agent_id, agent_name, prompt_id, model_id, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		a.ID.String(), a.Name, promptID.String(), a.ModelID.String(), now); err != nil {
		return fmt.Errorf("insert agent: %w", err)
	}
	if err := replaceAgentTools(ctx, tx, a.ID, a.ToolNames); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create agent: %w", err)
	}
	return nil
}

// GetAgent loads an agent definition joined with its prompt template.
func GetAgent(ctx context.Context, db *sql.DB, id uuid.UUID) (Agent, error) {
	var (
		a           Agent
		idString    string
		modelIDStr  string
		promptIDStr string
		updatedAt   sql.NullString
	)
	err := db.QueryRowContext(ctx, `
		SELECT a.agent_id, a.agent_name, a.model_id, a.prompt_id,
			pt.template_body, pt.params_schema, pt.version,
			a.created_at, a.updated_at
		FROM agents a
		JOIN prompt_templates pt ON pt.prompt_id = a.prompt_id
		WHERE a.agent_id = ? AND a.deleted_at IS NULL`, id.String()).Scan(
		&idString, &a.Name, &modelIDStr, &promptIDStr,
		&a.TemplateBody, &a.ParamsSchema, &a.Version,
		&a.CreatedAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Agent{}, ErrAgentNotFound
	}
	if err != nil {
		return Agent{}, fmt.Errorf("query agent: %w", err)
	}
	a.UpdatedAt = updatedAt.String

	a.ID, err = uuid.Parse(idString)
	if err != nil {
		return Agent{}, fmt.Errorf("parse agent id: %w", err)
	}
	a.ModelID, err = uuid.Parse(modelIDStr)
	if err != nil {
		return Agent{}, fmt.Errorf("parse model id: %w", err)
	}
	a.PromptID, err = uuid.Parse(promptIDStr)
	if err != nil {
		return Agent{}, fmt.Errorf("parse prompt id: %w", err)
	}

	toolNames, err := agentToolNames(ctx, db, a.ID)
	if err != nil {
		return Agent{}, err
	}
	a.ToolNames = toolNames
	return a, nil
}

// ListAgents returns every live agent definition, oldest first.
func ListAgents(ctx context.Context, db *sql.DB) ([]Agent, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT agent_id FROM agents
		WHERE deleted_at IS NULL
		ORDER BY created_at, agent_id`)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var idString string
		if err := rows.Scan(&idString); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan agent id: %w", err)
		}
		id, err := uuid.Parse(idString)
		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("parse agent id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate agents: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close agent rows: %w", err)
	}

	agents := make([]Agent, 0, len(ids))
	for _, id := range ids {
		a, err := GetAgent(ctx, db, id)
		if err != nil {
			return nil, fmt.Errorf("load listed agent: %w", err)
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// UpdateAgent replaces an agent's name, model, prompt template, and tool set.
func UpdateAgent(ctx context.Context, db *sql.DB, a Agent) error {
	if a.ID == (uuid.UUID{}) {
		return fmt.Errorf("update agent: id is required")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update agent: %w", err)
	}
	defer rollbackAgentTx(tx)

	var promptIDStr string
	err = tx.QueryRowContext(ctx, `
		SELECT prompt_id FROM agents WHERE agent_id = ? AND deleted_at IS NULL`,
		a.ID.String()).Scan(&promptIDStr)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAgentNotFound
	}
	if err != nil {
		return fmt.Errorf("query agent prompt id: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE prompt_templates
		SET template_body = ?, params_schema = ?, version = ?
		WHERE prompt_id = ?`,
		a.TemplateBody, a.ParamsSchema, versionOrDefault(a.Version), promptIDStr); err != nil {
		return fmt.Errorf("update prompt template: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `
		UPDATE agents
		SET agent_name = ?, model_id = ?, updated_at = ?
		WHERE agent_id = ? AND deleted_at IS NULL`,
		a.Name, a.ModelID.String(), now, a.ID.String())
	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check updated agent: %w", err)
	}
	if affected == 0 {
		return ErrAgentNotFound
	}
	if err := replaceAgentTools(ctx, tx, a.ID, a.ToolNames); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update agent: %w", err)
	}
	return nil
}

// DeleteAgent soft-deletes an agent definition.
func DeleteAgent(ctx context.Context, db *sql.DB, id uuid.UUID) error {
	result, err := db.ExecContext(ctx, `
		UPDATE agents SET deleted_at = ?
		WHERE agent_id = ? AND deleted_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339Nano), id.String())
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted agent: %w", err)
	}
	if affected == 0 {
		return ErrAgentNotFound
	}
	return nil
}

// replaceAgentTools resets agentID's agent_tools rows to exactly toolNames,
// resolved against the tools table by tool_name.
func replaceAgentTools(ctx context.Context, tx *sql.Tx, agentID uuid.UUID, toolNames []string) error {
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM agent_tools WHERE agent_id = ?`, agentID.String()); err != nil {
		return fmt.Errorf("delete agent tools: %w", err)
	}
	for _, name := range toolNames {
		var toolID string
		err := tx.QueryRowContext(ctx,
			`SELECT tool_id FROM tools WHERE tool_name = ?`, name).Scan(&toolID)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("tool %q: %w", name, ErrToolNotFound)
		}
		if err != nil {
			return fmt.Errorf("query tool id: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO agent_tools (agent_id, tool_id) VALUES (?, ?)`,
			agentID.String(), toolID); err != nil {
			return fmt.Errorf("insert agent tool: %w", err)
		}
	}
	return nil
}

// agentToolNames returns the tool_name of every tool linked to agentID via
// agent_tools, ordered by name. An agent with no linked tools returns nil.
func agentToolNames(ctx context.Context, db *sql.DB, agentID uuid.UUID) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT t.tool_name
		FROM agent_tools at
		JOIN tools t ON t.tool_id = at.tool_id
		WHERE at.agent_id = ?
		ORDER BY t.tool_name`, agentID.String())
	if err != nil {
		return nil, fmt.Errorf("query agent tool names: %w", err)
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
			return nil, fmt.Errorf("scan agent tool name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent tool names: %w", err)
	}
	return names, nil
}

func rollbackAgentTx(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		slog.Error("rollback agent transaction", "error", err)
	}
}

func versionOrDefault(version int) int {
	if version == 0 {
		return 1
	}
	return version
}
