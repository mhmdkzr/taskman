# list

Lists every task.

`list` performs no transition: it reads every id via `store.List`, then each task via
`store.Read`. An empty store returns an empty slice, not an error.

## CLI

```bash
taskman list
```

Prints one line per task as `id - title - [state]`, a `[]json.Document` array with `--json`, or a
Markdown document per task with `--md`.

## MCP

`task_list`, returning `{"tasks": [...]}` where each element is a `json.Document`. The tasks are
wrapped in an object because MCP structured content must be a JSON object, not a bare array.
