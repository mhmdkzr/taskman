{{ if .Trunk -}}
This task was created with --trunk: branch {{ .Branch }} is already the target, so there's nothing to merge. Report it with:
    task merge {{ .TaskID }} [--commit <hash>]
{{- else -}}
Merge branch {{ .Branch }} into the base branch yourself (from {{ .Worktree }} or the main checkout), then report it with:
    task merge {{ .TaskID }} [--commit <hash>]
{{- end }}
