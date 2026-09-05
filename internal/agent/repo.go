package agent

import (
	"context"
	"database/sql"
	"fmt"
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
