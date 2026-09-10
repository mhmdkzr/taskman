# specification

`taskman specified <id> --result <text> --done-when <text>` reports a typed
`SpecificationSubmitted` event. The pure workflow accepts it only in `specify`,
records the specification and acceptance criteria, and advances to
`specification_review`. A human then runs `taskman specification approved` or
`taskman specification rejected` before implementation can begin. CLI and MCP
are thin adapters over the same function.
