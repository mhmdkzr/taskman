# escalate

`taskman escalate <id> --stage <stage> --reason <text>` suspends a nonterminal,
nonblocked task. The pure workflow records the current state as `resume_state`
and moves to `blocked`. Terminal tasks and repeated escalation are rejected.
CLI and MCP share the same event adapter.
