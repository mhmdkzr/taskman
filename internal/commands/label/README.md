# label

Groups a task's label-editing actions: `label add` and `label remove`.

Labels are plain metadata (`Definition.Labels`), not part of the workflow, so both commands apply
via a global transition (`EventLabelsUpdated`) from any non-terminal state without changing the
task's current state - unlike every other reporting command, they carry no destination state of
their own.

## CLI

```bash
taskman label add --id <id> --label priority=high --label module=wallet
taskman label remove --id <id> --key priority
```

## MCP

`task_label_add`, `task_label_remove`.
