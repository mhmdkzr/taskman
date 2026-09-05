package tools

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/app/config"
)

type (
	Deps struct {
		DB         *sql.DB
		SessionID  sessions.SessionID
		Config     config.Config
		Configured map[ToolName]goai.Tool
	}

	ToolName    = string
	Constructor func(Deps) goai.Tool
	Registry    map[ToolName]Constructor
)

var ErrToolNotFound = errors.New("tool not found")

// Resolve builds the goai.Tool for each name using deps, in order. It fails
// on the first name with no registered constructor.
func (r Registry) Resolve(names []ToolName, deps Deps) ([]goai.Tool, error) {
	resolved := make([]goai.Tool, len(names))
	for i, name := range names {
		ctor, ok := r[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
		}
		resolved[i] = ctor(deps)
	}
	return resolved, nil
}
