# Task {{.ID}}

**State:** {{.State}}
**Next:** {{.Instruction.Action}} ({{.Instruction.State}})

## Definition

**Title:** {{if .Definition.Title}}{{.Definition.Title}}{{else}}_(none)_{{end}}

**Description:** {{.Definition.Description}}

{{if .Definition.Labels}}**Labels:**

{{range $key, $value := .Definition.Labels}}- {{$key}}: {{$value}}
{{end}}{{end}}
## History

| At | State |
| --- | --- |
{{range .StateHistory}}| {{.At.Format "2006-01-02 15:04:05 MST"}} | {{.State}} |
{{end}}
{{with .Specification}}
## Specification

**Plan:** {{.Plan}}

### Automated review

{{if .Review.Agent.Required}}Required.
{{template "autofix" .Review.Agent.AutoFix}}{{template "agentresults" .Review.Agent.Results}}{{else}}Not required.
{{end}}
### Human review

{{if .Review.Human.Required}}Required.
{{template "autofix" .Review.Human.AutoFix}}{{template "humanresults" .Review.Human.Results}}{{else}}Not required.
{{end}}
{{end}}
{{with .Implementation}}
## Implementation

**Worktree:** {{.Git.Worktree}}
**Branch:** {{.Git.Branch}}

{{if .Git.Commits}}**Commits:**

{{range .Git.Commits}}- `{{.Hash}}` {{.Message}} ({{.At.Format "2006-01-02 15:04:05 MST"}})
{{end}}
{{end}}{{with .Git.Merge}}**Merge:** into `{{.Target}}` at `{{.Commit}}` ({{.At.Format "2006-01-02 15:04:05 MST"}})

{{end}}### Verification

{{if .Verification.Tests.Unit}}- unit tests required
{{end}}{{if .Verification.Tests.Integration}}- integration tests required
{{end}}{{if .Verification.Tests.EndToEnd}}- end-to-end tests required
{{end}}{{if .Verification.Linters}}- linters required
{{end}}{{if not .Verification.Tests.Unit}}{{if not .Verification.Tests.Integration}}{{if not .Verification.Tests.EndToEnd}}{{if not .Verification.Linters}}- not required
{{end}}{{end}}{{end}}{{end}}
{{template "autofix" .Verification.AutoFix}}{{if .Verification.Attempts}}
**Attempts:**

{{range .Verification.Attempts}}- {{if .Passed}}passed{{else}}failed{{end}} ({{.At.Format "2006-01-02 15:04:05 MST"}}){{if .Output}}: {{.Output}}{{end}}
{{template "checks" .Checks}}{{end}}
{{end}}
### Automated review

{{if .Review.Agent.Required}}Required.
{{template "autofix" .Review.Agent.AutoFix}}{{template "agentresults" .Review.Agent.Results}}{{else}}Not required.
{{end}}
### Human review

{{if .Review.Human.Required}}Required.
{{template "autofix" .Review.Human.AutoFix}}{{template "humanresults" .Review.Human.Results}}{{else}}Not required.
{{end}}
{{end}}
{{with .Blocked}}
## Blocked

- **Stage:** {{.Stage}}
- **Reason:** {{.Reason}}
{{end}}
{{with .Abandoned}}
## Abandoned

- **Reason:** {{.Reason}}
- **At:** {{.At.Format "2006-01-02 15:04:05 MST"}}
{{end}}
{{define "autofix"}}{{if .Enabled}}Auto-fix enabled{{if .MaxRounds}}, max {{.MaxRounds}} round(s){{end}}{{if .UseSubagent}}, using a subagent{{end}}.
{{end}}{{end}}
{{define "checks"}}{{if .Unit}}  - unit: {{.Unit}}
{{end}}{{if .Integration}}  - integration: {{.Integration}}
{{end}}{{if .EndToEnd}}  - end-to-end: {{.EndToEnd}}
{{end}}{{if .Linters}}  - linters: {{.Linters}}
{{end}}{{end}}
{{define "agentresults"}}{{if .}}**Results:**

{{range .}}- {{if .Approved}}approved{{if .Comment}}: {{.Comment}}{{end}}{{else}}rejected{{range .Findings}}
  - **{{.Location}}:** {{.Detail}}{{end}}{{end}} ({{.At.Format "2006-01-02 15:04:05 MST"}})
{{end}}
{{end}}{{end}}
{{define "humanresults"}}{{if .}}**Results:**

{{range .}}- {{if .Approved}}approved{{else}}rejected{{end}}{{if .Comment}}: {{.Comment}}{{end}} ({{.At.Format "2006-01-02 15:04:05 MST"}})
{{end}}
{{end}}{{end}}
