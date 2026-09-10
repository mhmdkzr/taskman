# `internal/task`

The pure functional core for taskman's single compiled coding workflow.

`Task.State` is the workflow's one authoritative position. `definition.go`
declares every state, its instruction, and the events it accepts. `Apply`
evaluates a typed event against that definition and returns a new `Task`;
it never mutates its input or performs I/O. `Next` projects the instruction
for the current state from the same definition.

Events carry all external facts and timestamps into the core. Reducers update
a private clone, routes select the destination, and `Validate` checks the
resulting task invariants. Terminal states accept no further events. Git,
filesystem access, clocks, CLI/MCP types, and prompt rendering do not belong
in this package.

The workflow is intentionally concrete rather than a generic framework. A
second real workflow should exist before common workflow machinery is
extracted.

Persistence lives in `internal/taskstore`, Git execution in
`internal/gitclient`, and ID generation in `internal/taskid`.
