# abandoned

Ends a task unsuccessfully from any non-terminal state.

`abandoned` is a global transition landing on the terminal `abandoned` state; its reducer clears
any blockage. `--reason` is free text - there is no separate "kind" of failure to pick from. This
is a human-authorized decision.

## CLI

```bash
taskman abandoned --id <id> --reason "superseded by a different approach"
```

## MCP

`task_abandoned`.
