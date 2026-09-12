{{define "specify"}}Draft a specification for this task.

**Title:** {{if .Task.Definition.Title}}{{.Task.Definition.Title}}{{else}}_(none)_{{end}}
**Description:** {{.Task.Definition.Description}}
{{if .Task.Definition.Labels}}
**Labels:**
{{range $key, $value := .Task.Definition.Labels}}- {{$key}}: {{$value}}
{{end}}{{end}}{{if .Reason}}
**Address this feedback:**
{{.Reason}}
{{end}}
Report it with `specified`.
{{end}}
{{define "specification_review"}}This task's specification is awaiting review.

**Plan:**
{{.Task.Specification.Plan}}
{{if .Task.Specification.Review.Agent.Required}}
Automated review is required.{{end}}{{if .Task.Specification.Review.Human.Required}}
Human review is required.{{end}}
{{end}}
{{define "implement"}}Implement this task per its specification.

**Plan:**
{{.Task.Specification.Plan}}

Report completion with `implemented`.
{{end}}
{{define "verify"}}Run this implementation's required checks.

**Worktree:** {{.Task.Implementation.Git.Worktree}}
**Branch:** {{.Task.Implementation.Git.Branch}}

Required checks:
{{if .Task.Implementation.Verification.Tests.Unit}}- unit
{{end}}{{if .Task.Implementation.Verification.Tests.Integration}}- integration
{{end}}{{if .Task.Implementation.Verification.Tests.EndToEnd}}- end-to-end
{{end}}{{if .Task.Implementation.Verification.Linters}}- linters
{{end}}
Report the results with `verified`.
{{end}}
{{define "fix_verification_failure"}}Fix the reported verification failure, then re-run verification.

{{.Reason}}
{{end}}
{{define "fix_automated_review_findings"}}Fix the automated review's findings, then re-run verification.

{{.Reason}}
{{end}}
{{define "fix_human_review_findings"}}Fix the human review's findings, then re-run verification.

{{.Reason}}
{{end}}
{{define "automated_review"}}Perform an automated review of this implementation against its specification.

**Plan:**
{{.Task.Specification.Plan}}

Report the review with `implementation review agent approved` or `implementation review agent rejected`.
{{end}}
{{define "commit"}}Commit this implementation.

**Worktree:** {{.Task.Implementation.Git.Worktree}}
**Branch:** {{.Task.Implementation.Git.Branch}}

Report the commit with `committed`.
{{end}}
{{define "human_review"}}This task's implementation is awaiting human review.

**Branch:** {{.Task.Implementation.Git.Branch}}{{if .LastCommitHash}}
**Commit:** `{{.LastCommitHash}}`{{end}}
{{end}}
{{define "merge"}}Merge this implementation into its target branch.

**Branch:** {{.Task.Implementation.Git.Branch}}

Report the merge with `merged --target <branch>`.
{{end}}
{{define "blocked"}}This task is blocked.
{{if .Task.Blocked}}
**Stage:** {{.Task.Blocked.Stage}}
**Reason:** {{.Task.Blocked.Reason}}
{{end}}
{{end}}
{{define "completed"}}This task is complete. No further action applies.
{{end}}
{{define "abandoned"}}This task was abandoned.
{{if .Task.Abandoned}}
**Reason:** {{.Task.Abandoned.Reason}}
{{end}}
{{end}}
