# implement

`taskman implement <id>` reports that the current implementation attempt is
ready for checks. The command applies an `ImplementationCompleted` event; the
compiled workflow accepts it only in `implement` and advances to `verify`.
CLI and MCP share this behavior.
