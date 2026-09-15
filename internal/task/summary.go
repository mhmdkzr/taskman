package task

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
)

//go:embed task_summary.md
var taskSummaryFile embed.FS

var taskSummaryTmpl = template.Must(template.New("task_summary.md").
	ParseFS(taskSummaryFile, "task_summary.md"))

// TaskSummary is the default (non-JSON) CLI output for any command that
// returns a Task - the one prompt template genuinely shared by every
// command slice, so it lives here rather than in any one of them.
type TaskSummary struct {
	TaskID      string
	Title       string
	State       string
	Instruction string
}

func (p TaskSummary) Render() string {
	var b strings.Builder
	if err := taskSummaryTmpl.Execute(&b, p); err != nil {
		panic(fmt.Sprintf("support: render task_summary: %v", err))
	}
	return strings.TrimRight(b.String(), "\n")
}
