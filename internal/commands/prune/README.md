# prune

Permanently deletes every task in the `completed` state, along with its event log.

`prune` lists every task via `store.List`, replays each one via `store.Read`, and deletes the ones
whose current state is `completed`. Non-completed tasks - including `abandoned` - are left
untouched. A task whose events can no longer be replayed has no knowable state, so it is never
pruned and the replay error is returned instead.

Deletion is irreversible. The command reports which task ids were pruned (or, with `--dry-run`,
which *would* be), rendered as JSON (`--json`), Markdown (`--md`), or a plain summary.

## CLI

```bash
taskman prune
taskman prune --dry-run
```

## MCP

`task_prune` - takes an optional `dry-run` boolean and returns
`{"pruned": ["<id>", ...], "dry-run": false}`.
