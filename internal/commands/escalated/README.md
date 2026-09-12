# escalated

Blocks a task after dispatched work gives up.

`escalated` is a global transition: it applies from any non-terminal, non-blocked state (guarded by
`taskIsNotBlocked`) and lands on `blocked`, whose instruction is `wait`. It records the stage and
reason. There is no resume command. Use it for a worker giving up, not for an ordinary failed check
that can be retried.

## CLI

```bash
taskman escalated --id <id> --stage implementation --reason "requirements are ambiguous"
```

## MCP

`task_escalated`.
