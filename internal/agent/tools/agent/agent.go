package agent

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/app/config"
)

// deps bundles what execute needs to dispatch a named agent as a new,
// child session of the session that's calling this tool.
type deps struct {
	DB              *sql.DB
	Cfg             config.ProviderConfig
	Registry        tools.Registry
	ParentSessionID sessions.SessionID
}

// execute always dispatches a subagent, so ParentSessionID is required: this
// tool is only ever called from within an already-running session, never
// used to bootstrap a top-level one.
func execute(ctx context.Context, d deps, in input) (output, error) {
	agentCfg, err := sessions.AgentByName(ctx, d.DB, in.AgentName)
	if err != nil {
		return output{}, fmt.Errorf("agent: resolve %q: %w", in.AgentName, err)
	}

	// The child session must exist before its tools are resolved so its ID can
	// be supplied to tools that need session context.
	parent := d.ParentSessionID
	childID, err := sessions.Create(ctx, d.DB, d.Cfg, in.AgentName, in.Params, &parent)
	if err != nil {
		return output{}, fmt.Errorf("agent: %w", err)
	}

	toolNames, err := sessions.ToolNamesForAgent(ctx, d.DB, agentCfg.AgentID)
	if err != nil {
		return output{}, fmt.Errorf("agent: %w", err)
	}
	resolved, err := d.Registry.Resolve(toolNames, tools.Deps{
		DB:        d.DB,
		SessionID: childID,
	})
	if err != nil {
		return output{}, fmt.Errorf("agent: %w", err)
	}

	result, err := sessions.Run(ctx, d.DB, d.Cfg, childID, in.Task, resolved)
	if err != nil {
		return output{}, fmt.Errorf("agent: %w", err)
	}

	return output{
		SessionID: childID.String(),
		Result:    result.Text,
	}, nil
}
