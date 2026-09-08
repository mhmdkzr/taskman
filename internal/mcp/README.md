# `internal/mcp`

The MCP frontend's assembler - not a command itself, so it lives outside `internal/commands`.
`server.go`'s `NewServer` builds an `*mcp.Server` (from `github.com/modelcontextprotocol/go-sdk/mcp`)
and calls each wired-up slice's own `RegisterMCP` (under `internal/commands/<slice>/mcp.go`) to add
that slice's tool - `task_get`, `task_create`, and so on. The `mcp` CLI command itself
(`internal/commands/mcp/cmd.go`) is a regular slice like any other: it reads the root
`--tasks-dir`/`--git-dir`/`--worktrees-dir` flags, calls `NewServer` from this package, and serves
the result over stdio.

Each tool's input/output types and error mapping live next to the slice they belong to, not here:
this package only mounts them. A tool's input type is that slice's own `Request` struct (defined
in `<name>.go`, shared with the CLI's `cmd.go`), tagged with `json`/`jsonschema` so the SDK can
infer its schema directly - no separate MCP-only input type to keep in sync. Where `Request` has
fields worth checking (a required field, an id), the slice also defines a `func (r Request)
validate() error` next to it, called once at the top of the domain function itself - so both
`cmd.go` and `mcp.go` get identical validation for free without either calling it directly. A tool
handler returning a non-nil `error` is enough beyond that - the SDK's `AddTool` marks the result
`IsError` and surfaces the error text to the client automatically, so there's no CLI-style
exit-code mapping to do here.

See `internal/commands/mcp` for the `mcp` CLI command that serves this package's server over
stdio.

Every slice has an MCP tool (`internal/commands/<slice>/mcp.go`, registered here in `NewServer`),
following the same shape: `RegisterMCP(server, ...)` wires `mcp.AddTool(server, mcpTool(),
mcpHandler(...))`, with the tool definition and handler factored into their own `mcpTool()`/
`mcpHandler()` functions rather than built inline.
