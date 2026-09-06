// Package delete provides the agent-definition deletion tool.
package delete

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
	Name        = "agent_delete"
	Description = "Delete an agent definition by ID."
)

type Input struct {
	ID string `json:"id" jsonschema:"description=Agent UUID."`
}

type Output struct {
	ID      uuid.UUID `json:"id"`
	Deleted bool      `json:"deleted"`
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
	if err := agentrepo.DeleteAgent(ctx, st.RW(), id); err != nil {
		return Output{}, fmt.Errorf("delete agent: %w", err)
	}
	return Output{ID: id, Deleted: true}, nil
}
