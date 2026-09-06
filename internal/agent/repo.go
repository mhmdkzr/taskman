package agent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"uuid"
)

func newToolID() string {
	return uuid.NewV7().String()
}

func upsertTools(ctx context.Context, db *sql.DB, seeds []toolSeed) error {
	for _, seed := range seeds {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO tools (tool_id, tool_name, tool_description, input_schema, output_schema)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (tool_name) DO UPDATE SET
				tool_description = EXCLUDED.tool_description,
				input_schema = EXCLUDED.input_schema,
				output_schema = EXCLUDED.output_schema`,
			newToolID(), seed.Name, seed.Description, string(seed.InputSchema), string(seed.OutputSchema)); err != nil {
			return fmt.Errorf("upsert tool %q: %w", seed.Name, err)
		}
	}
	return nil
}

func createAgent(ctx context.Context, db *sql.DB, modelName, agentName, prompt string, seeds []toolSeed) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			slog.Error("rollback agent transaction", "error", rollbackErr)
		}
	}()

	var modelID string
	if err := tx.QueryRowContext(ctx, `
		SELECT m.model_id
		FROM models m
		JOIN model_providers p ON p.provider_id = m.provider_id
		WHERE p.provider_name = 'opencode' AND m.model_name = ?`, modelName).Scan(&modelID); err != nil {
		return fmt.Errorf("resolve model %q: %w", modelName, err)
	}

	promptID := uuid.NewV7().String()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO prompt_templates (prompt_id, prompt_name, template_body, params_schema, version)
		VALUES (?, ?, ?, '{}', 1)`, promptID, agentName, prompt); err != nil {
		return fmt.Errorf("insert prompt: %w", err)
	}
	agentID := uuid.NewV7().String()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO agents (agent_id, agent_name, prompt_id, model_id)
		VALUES (?, ?, ?, ?)`, agentID, agentName, promptID, modelID); err != nil {
		return fmt.Errorf("insert agent: %w", err)
	}
	for _, seed := range seeds {
		var toolID string
		if err := tx.QueryRowContext(ctx,
			`SELECT tool_id FROM tools WHERE tool_name = ?`, seed.Name,
		).Scan(&toolID); err != nil {
			return fmt.Errorf("resolve tool %q: %w", seed.Name, err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO agent_tools (agent_id, tool_id) VALUES (?, ?)`, agentID, toolID); err != nil {
			return fmt.Errorf("link tool %q: %w", seed.Name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
