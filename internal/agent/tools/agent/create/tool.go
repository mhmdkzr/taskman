// Package create provides the agent-definition creation tool.
package create

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
	Name        = "agent_create"
	Description = "Create a named agent definition: its model, prompt template, and tools."
)

type Input struct {
	Name         string   `json:"name" jsonschema:"description=Unique agent name, e.g. explore, review, commit."`
	ModelID      string   `json:"model_id" jsonschema:"description=UUID of the model row the agent runs on."`
	TemplateBody string   `json:"template_body" jsonschema:"description=System prompt template body."`
	ParamsSchema string   `json:"params_schema" jsonschema:"description=JSON schema describing the template's params."`
	Version      int      `json:"version,omitempty" jsonschema:"description=Prompt template version, defaults to 1."`
	ToolNames    []string `json:"tool_names,omitempty" jsonschema:"description=Names of tools the agent may call."`
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
	if strings.TrimSpace(in.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(in.ModelID) == "" {
		return errors.New("model_id is required")
	}
	if _, err := uuid.Parse(in.ModelID); err != nil {
		return fmt.Errorf("invalid model_id: %w", err)
	}
	if strings.TrimSpace(in.TemplateBody) == "" {
		return errors.New("template_body is required")
	}
	if strings.TrimSpace(in.ParamsSchema) == "" {
		return errors.New("params_schema is required")
	}
	if in.Version < 0 {
		return errors.New("version must not be negative")
	}
	return nil
}

func execute(ctx context.Context, st *store.Store, in Input) (Output, error) {
	if st == nil {
		return Output{}, errors.New("database is required")
	}
	modelID, err := uuid.Parse(in.ModelID)
	if err != nil {
		return Output{}, fmt.Errorf("invalid model_id: %w", err)
	}

	a := agentrepo.Agent{
		ID:           uuid.NewV7(),
		Name:         in.Name,
		ModelID:      modelID,
		TemplateBody: in.TemplateBody,
		ParamsSchema: in.ParamsSchema,
		Version:      in.Version,
		ToolNames:    in.ToolNames,
	}
	if err := agentrepo.CreateAgent(ctx, st.RW(), a); err != nil {
		return Output{}, fmt.Errorf("create agent: %w", err)
	}

	created, err := agentrepo.GetAgent(ctx, st.RO(), a.ID)
	if err != nil {
		return Output{}, fmt.Errorf("get created agent: %w", err)
	}
	return Output{Agent: created}, nil
}
