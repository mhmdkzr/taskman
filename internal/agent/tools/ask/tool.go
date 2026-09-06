// Package ask implements the user-question tool: the agent can pause a turn
// to ask the human a clarifying question, and resumes once they answer it
// through the web UI.
package ask

import (
	"context"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "ask"
	Description = "Ask the user a clarifying question and wait for their answer before continuing. " +
		"Use this only when you are genuinely blocked by ambiguity you cannot resolve from context or " +
		"by a decision only the user can make - not for routine confirmations or questions you could " +
		"answer yourself by reading more of the codebase."
)

// Input is one question, optionally with predefined choices.
type Input struct {
	Question    string   `json:"question"               jsonschema:"description=The full question to ask the user."`
	Options     []Option `json:"options,omitempty"      jsonschema:"description=Optional predefined choices. The user can always provide a custom answer instead."`
	MultiSelect bool     `json:"multi_select,omitempty" jsonschema:"description=Whether the user may select multiple predefined choices. Defaults to false for a single choice."`
}

// Option is one predefined choice offered alongside a free-form answer.
type Option struct {
	ID          int    `json:"id"                    jsonschema:"description=Stable numeric identifier for this choice. Returned in output.selected."`
	Label       string `json:"label"                 jsonschema:"description=The selectable answer text shown to the user."`
	Description string `json:"description,omitempty" jsonschema:"description=Optional context explaining this choice."`
}

// Output is the user's answer: the IDs of any predefined choices they
// selected, and/or free-form custom text.
type Output struct {
	Selected []int  `json:"selected,omitempty" jsonschema:"description=Numeric IDs of the selected predefined options."`
	Custom   string `json:"custom,omitempty"   jsonschema:"description=The user's free-form answer, if they gave one instead of or alongside a selection."`
}

// Tool asks its question as sessionID's currently running turn, blocking
// until a human answers it via the web UI (see execute).
func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d.Store, d.SessionID.String(), in)
	})
}

func (in Input) Validate() error {
	if strings.TrimSpace(in.Question) == "" {
		return fmt.Errorf("question is required")
	}
	if in.MultiSelect && len(in.Options) == 0 {
		return fmt.Errorf("multi_select requires at least one option")
	}
	return nil
}
