// Package view renders command output onto the CLI writer: every slice's
// Action renders a returned Task through PrintTask (--json for the JSON
// envelope, --md for the full Markdown document, otherwise a TaskSummary
// plus the task's rendered guidance and valid commands), and multi-task
// output through PrintJSON.
package view

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/view/instructions"
	jsonview "github.com/mhmdkzr/taskman/internal/task/view/json"
	mdview "github.com/mhmdkzr/taskman/internal/task/view/md"
)

func PrintTask(cmd *cli.Command, t task.Task) error {
	switch {
	case cmd.Bool("json"):
		doc, err := jsonview.FromTask(t)
		if err != nil {
			return fmt.Errorf("render task: %w", err)
		}
		return PrintJSON(cmd, doc)
	case cmd.Bool("md"):
		return PrintMarkdown(cmd, t)
	}
	instructions, err := instructions.Project(t)
	if err != nil {
		return fmt.Errorf("render task: %w", err)
	}
	w := cmd.Root().Writer
	if _, err := fmt.Fprintln(w, task.TaskSummary{
		TaskID:      t.ID.String(),
		Title:       t.Definition.Title,
		State:       t.State().String(),
		Instruction: fmt.Sprintf("%s (%s)", instructions.Action, instructions.State),
	}.Render()); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if _, err := fmt.Fprintf(w, "\n%s\n", instructions.Message); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	for _, command := range instructions.Commands {
		if _, err := fmt.Fprintf(w, "\n%s\n", command); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
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
