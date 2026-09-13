# delete

Permanently deletes a task and its entire event log from any state.

Unlike every other command, `delete` records no event and never replays the task's log: it removes
the task row and all of the task's events from the store in one transaction, keyed on identity
alone. Any task can therefore be deleted, including one whose events can no longer be replayed.
Once deleted, the task no longer appears in `list`, and `get`/`next` return "task not found".

This operation is irreversible. The command confirms with the deleted task's id, rendered as JSON
(`--json`) or a plain/Markdown line.

## CLI

```bash
taskman delete --id <id>
```

## MCP

`task_delete` - returns `{"id": "<deleted id>"}`.
