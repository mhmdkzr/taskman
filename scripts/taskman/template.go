package main

import (
	"fmt"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

const templateMD = `---
{{.FrontmatterYAML}}---

# {{.Title}}

## What
{{.What}}
{{if .How}}
## How
{{.How}}
{{end}}
## Why
{{.Why}}

## Done when
{{.DoneWhen}}
{{if .Questions}}
## Questions
{{range .Questions}}- {{.}}
{{end}}{{end}}`

var taskTemplate = template.Must(template.New("task").Parse(templateMD))

func renderTask(fm frontmatter, what, how, why, doneWhen string) (string, error) {
	fmYAML, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}
	data := struct {
		FrontmatterYAML string
		Title           string
		What            string
		How             string
		Why             string
		DoneWhen        string
		Questions       []string
	}{
		FrontmatterYAML: string(fmYAML),
		Title:           fm.Title,
		What:            strings.TrimSpace(what),
		How:             strings.TrimSpace(how),
		Why:             strings.TrimSpace(why),
		DoneWhen:        strings.TrimSpace(doneWhen),
		Questions:       fm.Questions,
	}
	var buf strings.Builder
	if err := taskTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
