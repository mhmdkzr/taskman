package implement

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed prompt.md
var promptFile embed.FS

var promptTmpl = template.Must(template.New("prompt.md").ParseFS(promptFile, "prompt.md"))

// Prompt is the dispatch prompt for the implementation stage - design.md §4.
type Prompt struct {
	Definition    string
	Specification string
	DoneWhen      string
	References    []string
}

// Render renders the prompt template with p's fields.
func (p Prompt) Render() string {
	var b strings.Builder
	if err := promptTmpl.Execute(&b, p); err != nil {
		panic(fmt.Sprintf("implement: render prompt: %v", err))
	}
	return strings.TrimRight(b.String(), "\n")
}
