# merged

Reads and records a merge already performed into a target branch.

Unlike `committed`, it does not read the task's worktree: it reads the `--target` ref in the
`--git-dir` repository via `git.Client.ReadBranchCommit`, then appends `MergeCompleted`. Perform
the actual `git merge` yourself first. From `merge`, this always completes the task.

## CLI

```bash
taskman merged --id <id> --target main
```

## MCP

`task_merged`.
