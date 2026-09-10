# create

`taskman create` validates task metadata, prepares a trunk checkout or isolated
Git worktree, and persists a task at the compiled workflow's initial
state. Supplying both a specification and acceptance criteria starts at
`implement`; otherwise it starts at `specify`.

Invoke it as `taskman create --definition <text>` with optional `--title`,
`--specification`, `--done-when`, `--trunk`, `--auto-approve`, labels, and
references. The CLI and MCP adapters call the same application function.
