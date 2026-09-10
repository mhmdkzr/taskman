Run the build checks yourself in {{ .Worktree }} (branch {{ .Branch }}). Scope them to what this
task actually changed (e.g. `go build`/`go test` on the affected package, not the whole repo). If
a check is inherently repo-wide (a formatter, a linter run with a fix flag), run `git status`
afterward - anything it touched outside this task's own change is not part of this task and must
not be committed with it.

Report each result with:
    verified {{ .TaskID }} --check <name>=<ok|error> ... [--output <text>]
