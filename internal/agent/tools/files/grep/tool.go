package grep

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "grep"
	Description = "Search files with a regular expression and return structured matching lines."
)

type Input struct {
	Pattern    string `json:"pattern" jsonschema:"description=Regular expression to search for."`
	Path       string `json:"path,omitempty" jsonschema:"description=File or directory to search. Defaults to the current directory."`
	Include    string `json:"include,omitempty" jsonschema:"description=Optional glob filter for file names."`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum number of matching lines. Defaults to 100."`
}

type Output struct {
	Matches   []string `json:"matches"`
	Truncated bool     `json:"truncated"`
}

func Tool() goai.Tool {
	return tools.Tool(Name, Description, execute)
}

func (in Input) Validate() error {
	if in.Pattern == "" {
		return errPatternRequired
	}
	return nil
}
