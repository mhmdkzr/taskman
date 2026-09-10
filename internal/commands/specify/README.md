# specified

`taskman specified <id> --result <text> --done-when <text>` reports a typed
`SpecificationSubmitted` event. The pure workflow accepts it only in `specify`,
records the specification and acceptance criteria, and advances to `implement`.
CLI and MCP are thin adapters over the same function.
