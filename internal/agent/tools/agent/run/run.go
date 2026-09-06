package run

import (
	"context"
	"fmt"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/store"
)

// deps bundles what execute needs to dispatch a named agent as a new,
// child session of the session that's calling this tool.
type deps struct {
	Store           *store.Store
	Cfg             config.ProviderConfig
	Registry        tools.Registry
	ParentSessionID sessions.SessionID
	Configured      map[string]goai.Tool
}

// execute always dispatches a subagent, so ParentSessionID is required: this
// tool is only ever called from within an already-running session, never
// used to bootstrap a top-level one.
func execute(ctx context.Context, d deps, in Input) (Output, error) {
	agentCfg, err := sessions.AgentByName(ctx, d.Store, in.AgentName)
	if err != nil {
		return Output{}, fmt.Errorf("agent: resolve %q: %w", in.AgentName, err)
	}

	// The child session must exist before its tools are resolved so its ID can
	// be supplied to tools that need session context.
	parent := d.ParentSessionID
	childID, err := sessions.Create(ctx, d.Store, d.Cfg, in.AgentName, in.Params, &parent)
	if err != nil {
		return Output{}, fmt.Errorf("agent: %w", err)
	}

	toolNames, err := sessions.ToolNamesForAgent(ctx, d.Store, agentCfg.AgentID)
	if err != nil {
		return Output{}, fmt.Errorf("agent: %w", err)
	}
	resolved, err := d.Registry.Resolve(toolNames, tools.Deps{
		Store:      d.Store,
		SessionID:  childID,
		Config:     config.Config{Provider: d.Cfg},
		Configured: d.Configured,
	})
	if err != nil {
		return Output{}, fmt.Errorf("agent: %w", err)
	}

	result, err := sessions.Run(ctx, d.Store, d.Cfg, childID, in.Task, resolved)
	if err != nil {
		return Output{}, fmt.Errorf("agent: %w", err)
	}

	return Output{
		SessionID: childID.String(),
		Result:    result.Text,
	}, nil
}
