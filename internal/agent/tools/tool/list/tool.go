// Package list provides the registered-tool listing tool.
package list

import (
	"context"
	"errors"
	"fmt"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	toolrepo "github.com/mhmdkzr/loop/internal/agent/tools/tool"
	"github.com/mhmdkzr/loop/internal/store"
)

const (
	Name        = "tool_list"
	Description = "List every registered tool."
)

type Input struct{}

type Output struct {
	Tools []toolrepo.Tool `json:"tools"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d.Store)
	})
}

func (in Input) Validate() error {
	return nil
}

func execute(ctx context.Context, st *store.Store) (Output, error) {
	if st == nil {
		return Output{}, errors.New("database is required")
	}
	toolList, err := toolrepo.ListTools(ctx, st.RO())
	if err != nil {
		return Output{}, fmt.Errorf("list tools: %w", err)
	}
	return Output{Tools: toolList}, nil
}
