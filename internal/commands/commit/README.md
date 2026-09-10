# committed

`taskman committed <id> [--commit <ref>]` is an imperative shell around the pure
workflow. It first verifies that `commit_recorded` is accepted, reads the real
commit from Git, and applies that fact as an event. Workflow routing selects
human review, merge, or terminal completion from `auto_approve` and trunk mode.
