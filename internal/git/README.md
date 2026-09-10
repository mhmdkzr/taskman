# `internal/gitclient`

The imperative Git adapter. It checks repository cleanliness, creates isolated
worktrees, observes commits, and commits terminal task-file bookkeeping. Git
values belong to `internal/task`; process execution and filesystem paths stay
here so the workflow core remains pure.
