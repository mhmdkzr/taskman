package skills

import (
	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = ""
	description = ""
)

type input struct{}

func Tool() goai.Tool {
	return tools.Tool(Name, description, execute)
}

func (input) Validate() error { return nil }
