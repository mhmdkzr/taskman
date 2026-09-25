# label remove

Deletes one or more of a task's labels.

Applies from any non-terminal state (labels are metadata, not workflow) and does not change the
task's current state. Removing a key that isn't present is not an error.

## CLI

```bash
taskman label remove --id <id> --key priority
```

## MCP

`task_label_remove`.
