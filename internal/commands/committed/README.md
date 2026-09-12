# committed

Reads and records the task's implementation worktree's current commit.

It does not take a hash or message: it reads `Implementation.Git.Worktree` (recorded by
`implemented`) through `git.Client.ReadCommit` and appends `CommitRecorded`. Create the actual
commit yourself first. From `commit`, the task routes to `human_review` if required, else
`merge`.

## CLI

```bash
taskman committed --id <id>
```

## MCP

`task_committed`.
