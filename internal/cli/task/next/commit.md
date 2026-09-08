Draft a commit message for this change, following the conventional-commit style, then run `git commit` yourself in the worktree to commit changes.

## Task
{{ .Title }}

## Specification
{{ .Specification }}

Stage only the files this task actually touched (`git add <path> ...`) - never a blanket `git add
-A` or `git commit -a`. Run `git status` first; if anything unrelated to this task shows as
changed, leave it out - including this task's own .tasks/{{ .TaskID }}.yaml, which will keep
changing after this commit (through review and merge) and gets committed separately once the task
is actually done.

taskman reads the commit's real message and hash back out of git itself once you report it - you don't need to repeat them.
