package tool

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"uuid"
)

var ErrToolNotFound = errors.New("tool not found")

// GetToolByName resolves a registered tool by its tool_name, which is unique
// per deployment and the identifier callers of this package's tools know.
func GetToolByName(ctx context.Context, db *sql.DB, name string) (Tool, error) {
	var (
		t        Tool
		idString string
	)
	err := db.QueryRowContext(ctx, `
		SELECT tool_id, tool_name, tool_description, input_schema, output_schema
		FROM tools
		WHERE tool_name = ?`, name).Scan(
		&idString, &t.Name, &t.Description, &t.InputSchema, &t.OutputSchema,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Tool{}, ErrToolNotFound
	}
	if err != nil {
		return Tool{}, fmt.Errorf("query tool: %w", err)
	}
	t.ID, err = uuid.Parse(idString)
	if err != nil {
		return Tool{}, fmt.Errorf("parse tool id: %w", err)
	}
	return t, nil
}

// ListTools returns every registered tool, ordered by name.
func ListTools(ctx context.Context, db *sql.DB) ([]Tool, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT tool_id, tool_name, tool_description, input_schema, output_schema
		FROM tools
		ORDER BY tool_name`)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close tool rows", "error", err)
		}
	}()

	var tools []Tool
	for rows.Next() {
		var (
			t        Tool
			idString string
		)
		if err := rows.Scan(&idString, &t.Name, &t.Description, &t.InputSchema, &t.OutputSchema); err != nil {
			return nil, fmt.Errorf("scan tool: %w", err)
		}
		t.ID, err = uuid.Parse(idString)
		if err != nil {
			return nil, fmt.Errorf("parse tool id: %w", err)
		}
		tools = append(tools, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tools: %w", err)
	}
	return tools, nil
}
