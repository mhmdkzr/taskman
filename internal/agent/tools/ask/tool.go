package ask

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = ""
	description = ""
)

type input struct {
	Question    string   `json:"question"               jsonschema:"description=The full question to ask the user."`
	Header      string   `json:"header"                 jsonschema:"description=A short label for the question, suitable for displaying in a compact UI."`
	Options     []option `json:"options,omitempty"      jsonschema:"description=Optional predefined choices. The user can always provide a custom answer."`
	MultiSelect bool     `json:"multi_select,omitempty" jsonschema:"description=Whether the user may select multiple predefined choices. Defaults to false for a single choice."`
}

type option struct {
	ID          int    `json:"id"                    jsonschema:"description=Stable numeric identifier for this choice. Returned in output.selected."`
	Label       string `json:"label"                 jsonschema:"description=The selectable answer text shown to the user."`
	Description string `json:"description,omitempty" jsonschema:"description=Optional context explaining this choice."`
}

type output struct {
	Selected []int  `json:"selected,omitempty" jsonschema:"description=Numeric IDs of the selected predefined options."`
	Custom   string `json:"custom,omitempty"`
}

func Tool() goai.Tool {
	return tools.Tool(Name, description, execute)
}

func (input) Validate() error { return nil }
