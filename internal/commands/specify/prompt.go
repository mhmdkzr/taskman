package specify

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed prompt.md
var promptFile embed.FS

var promptTmpl = template.Must(template.New("prompt.md").ParseFS(promptFile, "prompt.md"))

// Prompt is the dispatch prompt for the specification stage.
type Prompt struct {
	Definition string
	References []string
}

// Render renders the prompt template with p's fields.
func (p Prompt) Render() string {
	var b strings.Builder
	if err := promptTmpl.Execute(&b, p); err != nil {
		panic(fmt.Sprintf("specify: render prompt: %v", err))
	}
	return strings.TrimRight(b.String(), "\n")
}
