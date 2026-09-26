# Task {{.ID}}

**State:** {{.State}}
**Next:** {{.Instruction.Action}} ({{.Instruction.State}})

{{.Guidance.Message}}
{{if .Guidance.Commands}}
Valid commands:

{{range .Guidance.Commands}}- `{{.}}`
{{end}}{{end}}

## Definition

**Title:** {{.Definition.Title}}

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
{{if not $.Implementation}}
Verification checks, implementation review gates, and worktree/branch are fixed here and copied
forward automatically once `implemented` is reported - shown below as planned.

### Verification (planned)

{{template "verificationChecks" .Verification}}
{{template "autofix" .Verification.AutoFix}}
### Implementation review (planned)

{{if .ImplementationReview.Agent.Required}}Automated review required.
{{template "autofix" .ImplementationReview.Agent.AutoFix}}
{{end}}{{if .ImplementationReview.Human.Required}}Human review required.
{{template "autofix" .ImplementationReview.Human.AutoFix}}
{{end}}{{if not .ImplementationReview.Agent.Required}}{{if not .ImplementationReview.Human.Required}}Not required.
{{end}}{{end}}
{{if .Worktree.UseWorktree}}### Worktree (planned)

**Worktree:** {{.Worktree.Worktree}}
**Branch:** {{.Worktree.Branch}}
{{end}}{{end}}
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

{{template "verificationChecks" .Verification}}
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
{{define "verificationChecks"}}{{if .Tests.Unit}}- unit tests required
{{end}}{{if .Tests.Integration}}- integration tests required
{{end}}{{if .Tests.EndToEnd}}- end-to-end tests required
{{end}}{{if .Linters}}- linters required
{{end}}{{if not .Tests.Unit}}{{if not .Tests.Integration}}{{if not .Tests.EndToEnd}}{{if not .Linters}}- not required
{{end}}{{end}}{{end}}{{end}}{{end}}
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
