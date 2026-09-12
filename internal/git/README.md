# `internal/git`

The imperative Git adapter used by command shells to read facts from a repository. Process
execution and filesystem paths stay here so the pure `internal/task` core performs no filesystem
or process work.

`Client` wraps a repository root (the CLI's `--git-dir`) and exposes two reads:

- `ReadCommit(ctx, worktreeDir, ref)` - the hash and message of `ref` (default `HEAD`) in a
  specific worktree. Used by `committed` to read the task's recorded implementation worktree.
- `ReadBranchCommit(ctx, ref)` - the same, resolved in the client's own repository root. Used by
  `merged` to read the target branch after a merge, since that ref lives in the main repository
  rather than in the task's worktree.

Each read records the time taskman observed the commit, not the commit's own Git timestamp.
