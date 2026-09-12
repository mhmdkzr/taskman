# mcp

Serves taskman's operations as MCP tools over stdio.

`taskman mcp` opens the store at `--db` and a Git client at `--git-dir`, builds the server via
`internal/mcp`, and blocks serving MCP/stdio. The root flags are bound once at startup rather than
passed per tool call; every tool shares the same store and Git client for the server's lifetime.

## CLI

```bash
taskman mcp
```

## See also

- `internal/mcp` - the server assembler and tool conventions.
