// Package run implements the agent-dispatch tool.
package run

import (
	"context"
	"errors"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/store"
)

const (
	Name        = "agent"
	Description = "Dispatch a task to a named agent, running it as a new child session with that agent's model, system prompt, and tools. Returns the agent's final text result."
)

var (
	errAgentNameRequired = errors.New("agent_name is required")
	errTaskRequired      = errors.New("task is required")
)

type Input struct {
	AgentName string         `json:"agent_name"       jsonschema:"description=Name of the agent to dispatch, e.g. explore, review, commit."`
	Task      string         `json:"task"             jsonschema:"description=The task for the dispatched agent to perform."`
	Params    map[string]any `json:"params,omitempty" jsonschema:"description=Named values to render into the agent's prompt template."`
}

type Output struct {
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
}

// Tool returns the agent dispatch tool. db, cfg, and registry are the
// runtime dependencies used to create and run the dispatched session;
// parentSessionID scopes the new session to the caller's session.
func Tool(
	st *store.Store,
	cfg config.ProviderConfig,
	registry tools.Registry,
	parentSessionID sessions.SessionID,
	configured map[string]goai.Tool,
) goai.Tool {
	d := deps{
		Store:           st,
		Cfg:             cfg,
		Registry:        registry,
		ParentSessionID: parentSessionID,
		Configured:      configured,
	}
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d, in)
	})
}

func (in Input) Validate() error {
	if in.AgentName == "" {
		return errAgentNameRequired
	}
	if in.Task == "" {
		return errTaskRequired
	}
	return nil
}
