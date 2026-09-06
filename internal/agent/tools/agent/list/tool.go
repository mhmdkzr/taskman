// Package list provides the agent-definition listing tool.
package list

import (
	"context"
	"errors"
	"fmt"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	agentrepo "github.com/mhmdkzr/loop/internal/agent/tools/agent"
	"github.com/mhmdkzr/loop/internal/store"
)

const (
	Name        = "agent_list"
	Description = "List every agent definition."
)

type Input struct{}

type Output struct {
	Agents []agentrepo.Agent `json:"agents"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d.Store)
	})
}

func (in Input) Validate() error {
	return nil
}

func execute(ctx context.Context, st *store.Store) (Output, error) {
	if st == nil {
		return Output{}, errors.New("database is required")
	}
	agents, err := agentrepo.ListAgents(ctx, st.RO())
	if err != nil {
		return Output{}, fmt.Errorf("list agents: %w", err)
	}
	return Output{Agents: agents}, nil
}
