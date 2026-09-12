# list

Lists every task.

`list` performs no transition: it reads every id via `store.List`, then each task via
`store.Read`. An empty store returns an empty slice, not an error.

## CLI

```bash
taskman list
```

Prints a summary per task, a `[]json.Document` array with `--json`, or a Markdown document per
task with `--md`.

## MCP

`task_list`, returning `[]json.Document`.
