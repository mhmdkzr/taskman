package curl

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "curl"
	description = "Run curl with the supplied arguments and return its standard output. The curl executable must be available on the system."
)

const Description = description

type input struct {
	Args []string `json:"args" jsonschema:"description=Arguments to pass to curl, for example [\"-sS\", \"https://example.com\"]. Do not include the curl executable name."`
}

type Input = input

func Tool() goai.Tool {
	return tools.Tool(Name, description, execute)
}

func (input) Validate() error { return nil }
