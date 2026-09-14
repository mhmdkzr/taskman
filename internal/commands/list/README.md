# list

Lists tasks, optionally filtered by state and labels and sliced into a page.

`list` performs no transition: it reads every id via `store.List`, replays each task via
`store.Read`, keeps the matches, and slices the page. An empty result returns an empty slice, not an
error. Filtering happens after replay because a task's state is derived from its event log.

## Filters and pagination

- `--state <state>` keeps tasks in any of the given states; repeatable. An unknown state is
  malformed input (exit 2).
- `--label <key=value>` keeps tasks carrying every given label; repeatable.
- `--limit <n>` caps the page size (default 50); `0` means unlimited.
- `--offset <n>` skips that many matching tasks before the page starts.

The total number of matches (before slicing) is always reported, so a caller can paginate: when
`offset + len(page) < total` the human output prints a `... N more` footer, and `--json` carries
`total`.

## CLI

```bash
taskman list --state implement --label priority=high --limit 20 --offset 20
```

Prints one line per task as `id - title - [state]`, a `{tasks, total, limit, offset}` object with
`--json`, or a Markdown document per task with `--md`.

## MCP

`task_list`, taking the same `state`, `label`, `limit`, and `offset` inputs and returning
`{tasks, total, limit, offset}`. The tasks are wrapped in an object because MCP structured content
must be a JSON object, not a bare array. The MCP default for `limit` is `0` (unlimited); pass a
value to paginate.
