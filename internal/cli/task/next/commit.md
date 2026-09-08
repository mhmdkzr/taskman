Draft a commit message for this change, following the conventional-commit style, then run `git commit` yourself in the worktree to commit changes.

## Task
{{ .Title }}

## Specification
{{ .Specification }}

Stage only the files this task actually touched (`git add <path> ...`) - never a blanket `git add
-A` or `git commit -a`. Run `git status` first; if anything unrelated to this task shows as
changed, leave it out.

taskman reads the commit's real message and hash back out of git itself once you report it - you don't need to repeat them.
