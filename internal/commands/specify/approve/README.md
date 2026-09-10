# specification-approved

`taskman specification approved <id>` records a human approval for a drafted
specification. It is accepted only while the task is in `specification_review`
and advances the task to `implement`.

Use `--comment` to retain an optional approval note. The CLI and MCP adapters
call the same application function.
