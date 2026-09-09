{{ if .Trunk -}}
Task {{ .TaskID }} ('{{ .Title }}') is complete - already on branch {{ .Branch }} as {{ .CommitHash }} (created with --trunk, so no merge was needed).
{{- else -}}
Task {{ .TaskID }} ('{{ .Title }}') is complete - branch {{ .Branch }} merged as {{ .CommitHash }}.
{{- end }}
