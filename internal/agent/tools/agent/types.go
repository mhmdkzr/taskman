// Package agent implements CRUD tools for agent definitions, plus the
// agent-dispatch and agent-schedule tools in its subpackages.
package agent

import "uuid"

// Agent is a named agent definition: the model it runs on, the prompt
// template its system prompt is rendered from, and the tools it may call.
type Agent struct {
	ID           uuid.UUID
	Name         string
	ModelID      uuid.UUID
	PromptID     uuid.UUID
	TemplateBody string
	ParamsSchema string
	Version      int
	ToolNames    []string
	CreatedAt    string
	UpdatedAt    string
}
