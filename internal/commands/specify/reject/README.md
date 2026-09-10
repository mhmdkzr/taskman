# specification-rejected

`taskman specification rejected <id> --reason <text>` records a human
rejection for a drafted specification. It is accepted only while the task is
in `specification_review` and returns the task to `specify` for revision.

The rejection reason is retained in the task's specification-review history
and included in the next specification prompt. The CLI and MCP adapters call
the same application function.
