package prompts

import "embed"

//go:embed system-prompt.md
var systemPrompt embed.FS

//go:embed conventional-commits.md
var conventionalCommits embed.FS

//go:embed semantic-versioning.md
var semanticVersioning embed.FS

//go:embed execution-agent.md
var executionAgent embed.FS

//go:embed review-agent.md
var reviewAgent embed.FS

//go:embed commit-agent.md
var commitAgent embed.FS

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

// ExecutionAgent is the system prompt for the pipeline's execution agent: the
// role that implements a task in its own isolated worktree.
func ExecutionAgent() (string, error) {
	p, err := executionAgent.ReadFile("execution-agent.md")
	if err != nil {
		return "", err
	}
	return string(p), nil
}

// ReviewAgent is the system prompt for the pipeline's review agent: a
// separate, read-only-tooled session that checks the execution agent's work.
func ReviewAgent() (string, error) {
	p, err := reviewAgent.ReadFile("review-agent.md")
	if err != nil {
		return "", err
	}
	return string(p), nil
}

// CommitAgent is the system prompt for the pipeline's commit agent: a
// tool-less, single-shot role that writes the final commit message from the
// actual diff being committed.
func CommitAgent() (string, error) {
	p, err := commitAgent.ReadFile("commit-agent.md")
	if err != nil {
		return "", err
	}
	return string(p), nil
}
