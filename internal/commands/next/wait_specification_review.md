Task {{ .TaskID }} ('{{ .Title }}') is awaiting human approval of its drafted specification.

## Specification
{{ .Specification }}

## Done when
{{ .DoneWhen }}

Nothing proceeds until a human runs `specification approved {{ .TaskID }}` or `specification rejected {{ .TaskID }} --reason "..."`.
