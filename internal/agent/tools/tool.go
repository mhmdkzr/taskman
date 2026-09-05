// Package tools defines the shared agent-tool interfaces and registry.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zendev-sh/goai"
)

type Validatable interface {
	Validate() error
}

func Tool[In Validatable, Out any](
	name, description string,
	execute func(ctx context.Context, in In) (Out, error),
) goai.Tool {
	return goai.NewTool(name, description, func(ctx context.Context, in In) (string, error) {
		if err := in.Validate(); err != nil {
			return "", fmt.Errorf("validate tool input: %w", err)
		}

		result, err := execute(ctx, in)
		if err != nil {
			return "", fmt.Errorf("execute tool: %w", err)
		}

		encoded, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("encode error: %w", err)
		}
		return string(encoded), nil
	})
}
