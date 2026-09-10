# merged

`taskman merged <id> [--commit <hash>]` reports a caller-performed merge. It
applies `MergeCompleted` in the pure workflow, optionally updates the recorded
commit hash, persists `completed`, and commits the terminal task file as Git
bookkeeping. CLI and MCP call the same application function.
