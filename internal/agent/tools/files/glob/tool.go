package glob

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "glob_files"
	Description = "Find files matching a glob pattern (supports ** for recursive matching), most recently modified first."
)

type Input struct {
	Pattern string `json:"pattern"         jsonschema:"description=Glob pattern relative to path, e.g. **/*.go or internal/**/README.md."`
	Path    string `json:"path,omitempty"  jsonschema:"description=Directory to search from. Defaults to the current directory."`
	Limit   int    `json:"limit,omitempty" jsonschema:"description=Maximum number of matches to return. Defaults to 200."`
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
