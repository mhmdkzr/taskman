# verified

`taskman verified <id> --check <name>=<ok|error> ...` records one verification
attempt. The command converts the report to a typed event. The pure workflow
routes passing checks to automated review and failures to the appropriate fix
state, preserving human-review recovery semantics. CLI and MCP share it.
