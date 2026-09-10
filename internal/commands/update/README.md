# update

`taskman update <id>` patches non-workflow metadata such as title, labels,
references, trunk mode, and auto-approval. It uses the value-oriented locked
task store but deliberately emits no workflow event and never changes the
current state. CLI and MCP call the same function.
