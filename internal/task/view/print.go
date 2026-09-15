// Package view renders command output onto the CLI writer: every slice's
// Action renders a returned Task through PrintTask (--json for the JSON
// envelope, --md for the full Markdown document, a short TaskSummary
// otherwise), and multi-task output through PrintJSON.
package view

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
	jsonview "github.com/mhmdkzr/taskman/internal/task/view/json"
	mdview "github.com/mhmdkzr/taskman/internal/task/view/md"
)

func PrintTask(cmd *cli.Command, t task.Task) error {
	switch {
	case cmd.Bool("json"):
		return PrintJSON(cmd, jsonview.FromTask(t))
	case cmd.Bool("md"):
		return PrintMarkdown(cmd, t)
	}
	instruction := t.Instruction()
	if _, err := fmt.Fprintln(cmd.Root().Writer, task.TaskSummary{
		TaskID:      t.ID.String(),
		Title:       t.Definition.Title,
		State:       t.State().String(),
		Instruction: fmt.Sprintf("%s (%s)", instruction.Action, instruction.State),
	}.Render()); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// PrintMarkdown writes t to stdout as a Markdown document.
func PrintMarkdown(cmd *cli.Command, t task.Task) error {
	doc, err := mdview.RenderTask(t)
	if err != nil {
		return fmt.Errorf("render markdown: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.Root().Writer, strings.TrimRight(doc, "\n")); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// PrintJSON writes v to stdout as indented JSON.
func PrintJSON(cmd *cli.Command, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.Root().Writer, string(data)); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
