{{ if .Trunk -}}
Task {{ .TaskID }} ('{{ .Title }}') is complete - already on branch {{ .Branch }} as {{ .CommitHash }} (created with --trunk, so no merge was needed).
{{- else -}}
Task {{ .TaskID }} ('{{ .Title }}') is complete - branch {{ .Branch }} merged as {{ .CommitHash }}.
{{- end }}

This task's own task file (.tasks/{{ .TaskID }}.yaml under your --tasks-dir) changed throughout
this run and is likely still uncommitted. Commit it now on its own, as a small chore commit - e.g.
`chore(.tasks): record completion of {{ .TaskID }}` - separate from the code commit above.
