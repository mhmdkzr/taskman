# get

Reads one task's current state and instruction.

`get` performs no transition: it replays the task's event log through `store.Read` and returns the
resulting task. The output includes its derived `state` and `instruction`.

## CLI

```bash
taskman get --id <id>
```

`--id` is required. Prints a summary, the JSON envelope with `--json`, or a Markdown document with
`--md`.

## MCP

`task_get`.
