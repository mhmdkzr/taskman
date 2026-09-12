# specified

Records a task's drafted specification and which of its review gates are required.

`specified` builds a `SpecificationSubmitted` event and applies it via `store.Append` (validated by
`task.Apply` before persisting). From `specify` it routes to `specification_review` when either
review gate is required, otherwise straight to `implement`.

`specified` may be called again whenever the task is back in `specify` after a rejection: each
submission replaces the specification wholesale and clears prior review results, so its review
gates can still be reconfigured until the task leaves `specify`.

## Request

| Field | Flag | Required |
|---|---|---|
| `ID` | `--id` | yes |
| `Plan` | `--plan` | yes |
| `Review` | `--agent-review` / `--human-review` (+ auto-fix flags) | no |

The review flags are shared with `implemented` via `utils.ReviewFlags()`; see `internal/utils`.

## CLI

```bash
taskman specified --id <id> --plan "..." --agent-review
```

## MCP

`task_specified`.
