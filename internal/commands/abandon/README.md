# abandon

`taskman abandon <id> --reason <text>` applies an `Abandoned` event to any
nonterminal task, persists the terminal `abandoned` state, and commits the task
file as Git bookkeeping. Repeating it or abandoning a completed task is an
invalid transition. CLI and MCP share the same application function.
