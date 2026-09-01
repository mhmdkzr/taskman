package prompts

import "embed"

//go:embed system-prompt.md
var systemPrompt embed.FS

//go:embed conventional-commits.md
var conventionalCommits embed.FS

//go:embed semantic-versioning.md
var semanticVersioning embed.FS

func SystemPrompt() (string, error) {
	p, err := systemPrompt.ReadFile("system-prompt.md")
	if err != nil {
		return "", err
	}
	return string(p), nil
}

func ConventionalCommits() (string, error) {
	p, err := conventionalCommits.ReadFile("conventional-commits.md")
	if err != nil {
		return "", err
	}
	return string(p), nil
}

func SemanticVersioning() (string, error) {
	p, err := semanticVersioning.ReadFile("semantic-versioning.md")
	if err != nil {
		return "", err
	}
	return string(p), nil
}
