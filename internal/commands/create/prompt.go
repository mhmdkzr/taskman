package create

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed prompt.md
var promptFile embed.FS

var promptTmpl = template.Must(template.New("prompt.md").ParseFS(promptFile, "prompt.md"))

// Summary is task create's default (non-JSON) CLI output.
type Summary struct {
	TaskID   string
	Title    string
	Worktree string
	Branch   string
}

// Render renders the summary template with s's fields.
func (s Summary) Render() string {
	var b strings.Builder
	if err := promptTmpl.Execute(&b, s); err != nil {
		panic(fmt.Sprintf("create: render prompt: %v", err))
	}
	return strings.TrimRight(b.String(), "\n")
}
