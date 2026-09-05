package datetime

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "datetime"
	description = "Get the current date and time, or convert a Unix timestamp, in an IANA timezone. Defaults to UTC. Unix timestamps may be in seconds, milliseconds, microseconds, or nanoseconds."
)

const Description = description

type input struct {
	Timezone  string `json:"timezone,omitempty"       jsonschema:"description=IANA timezone name such as UTC or America/New_York (default UTC)."`
	Timestamp *int64 `json:"timestamp,omitempty"      jsonschema:"description=Optional Unix timestamp to convert. Use timestamp_unit to specify seconds, milliseconds, microseconds, or nanoseconds."`
	Unit      string `json:"timestamp_unit,omitempty" jsonschema:"description=Unit of timestamp: s, ms, us, or ns. If omitted, the unit is inferred from the magnitude."`
}

type (
	Input  = input
	Output = output
)

func Tool() goai.Tool {
	return tools.Tool(Name, description, execute)
}

func (input) Validate() error { return nil }
