Run the build checks yourself in {{ .Worktree }} (branch {{ .Branch }}), then report each result with:
    task verify {{ .TaskID }} --check <name>=<ok|error> ... [--output <text>]
