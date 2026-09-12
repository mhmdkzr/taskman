# create

Creates a new task in the `specify` state and returns its generated id.

`create` is a management command: it has no `TaskEvent` and does not go through `store.Append`. It
generates a UUIDv7 id and calls `store.Create`, which seeds the state history at
`task.StateSpecify` via `task.NewTask`. The new task's instruction is `dispatch` (draft and report
a specification with `specified`).

## Request

| Field | Flag | Required |
|---|---|---|
| `Title` | `--title` | no |
| `Description` | `--description` | yes |
| `Labels` | `--label k=v` (repeatable) | no |

## CLI

```bash
taskman create --description "add per-client rate limiting" --title "Rate limiting" --label priority=high
```

## MCP

`task_create`, taking `Request` as its typed input.
