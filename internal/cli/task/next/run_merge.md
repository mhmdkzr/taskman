Merge branch {{ .Branch }} into the base branch yourself (from {{ .Worktree }} or the main checkout), then report it with:
    task merge {{ .TaskID }} [--commit <hash>]
