package write

import (
	"context"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "notes_write"
	Description = "Create or update a persistent note and replace its links to other notes."
)

type Link struct {
	Name         string `json:"name"         jsonschema:"description=Name of the note to link to."`
	Relationship string `json:"relationship" jsonschema:"description=Optional description of the relationship between the notes."`
}

type Input struct {
	Name  string `json:"name"  jsonschema:"description=Unique name of the note."`
	Body  string `json:"body"  jsonschema:"description=Complete note content."`
	Links []Link `json:"links" jsonschema:"description=Notes to link to; omitted links are removed when updating an existing note."`
}

type Output struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Body  string `json:"body"`
	Links []Link `json:"links"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d, in)
	})
}

func (in Input) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(in.Body) == "" {
		return fmt.Errorf("body is required")
	}
	seen := make(map[string]struct{}, len(in.Links))
	for _, link := range in.Links {
		name := strings.TrimSpace(link.Name)
		if name == "" {
			return fmt.Errorf("link name is required")
		}
		if name == strings.TrimSpace(in.Name) {
			return fmt.Errorf("a note cannot link to itself")
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate link %q", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}
