// Package get provides the registered-tool lookup tool.
package get

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	toolrepo "github.com/mhmdkzr/loop/internal/agent/tools/tool"
	"github.com/mhmdkzr/loop/internal/store"
)

const (
	Name        = "tool_get"
	Description = "Get a registered tool by name."
)

type Input struct {
	Name string `json:"name" jsonschema:"description=Registered tool_name."`
}

type Output struct {
	Tool toolrepo.Tool `json:"tool"`
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
	return nil
}

func execute(ctx context.Context, st *store.Store, in Input) (Output, error) {
	if st == nil {
		return Output{}, errors.New("database is required")
	}
	t, err := toolrepo.GetToolByName(ctx, st.RO(), in.Name)
	if err != nil {
		return Output{}, fmt.Errorf("get tool: %w", err)
	}
	return Output{Tool: t}, nil
}
