// Package get provides the agent-definition lookup tool.
package get

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	agentrepo "github.com/mhmdkzr/loop/internal/agent/tools/agent"
	"github.com/mhmdkzr/loop/internal/store"
)

const (
	Name        = "agent_get"
	Description = "Get an agent definition by ID."
)

type Input struct {
	ID string `json:"id" jsonschema:"description=Agent UUID."`
}

type Output struct {
	Agent agentrepo.Agent `json:"agent"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d.Store, in)
	})
}

func (in Input) Validate() error {
	if strings.TrimSpace(in.ID) == "" {
		return errors.New("id is required")
	}
	if _, err := uuid.Parse(in.ID); err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	return nil
}

func execute(ctx context.Context, st *store.Store, in Input) (Output, error) {
	if st == nil {
		return Output{}, errors.New("database is required")
	}
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return Output{}, fmt.Errorf("invalid id: %w", err)
	}
	a, err := agentrepo.GetAgent(ctx, st.RO(), id)
	if err != nil {
		return Output{}, fmt.Errorf("get agent: %w", err)
	}
	return Output{Agent: a}, nil
}
