# label add

Sets or overwrites one or more of a task's labels.

Applies from any non-terminal state (labels are metadata, not workflow) and does not change the
task's current state.

## CLI

```bash
taskman label add --id <id> --label priority=high --label module=wallet
```

## MCP

`task_label_add`.
