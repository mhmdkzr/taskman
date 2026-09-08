Task {{ .TaskID }} ('{{ .Title }}') was abandoned: {{ .Reason }}

This task's own task file (.tasks/{{ .TaskID }}.yaml under your --tasks-dir) changed throughout
this run and is likely still uncommitted. Commit it now on its own, as a small chore commit - e.g.
`chore(.tasks): record abandonment of {{ .TaskID }}`.
