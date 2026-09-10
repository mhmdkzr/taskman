# skill

`taskman skill` prints the agent-facing Taskman driver skill embedded in the
binary. `SKILL.md` is the source of that output and is embedded verbatim, so an
agent can install or inspect the workflow guidance without access to this
repository.

The command writes only to stdout and has no MCP equivalent. Lifecycle
operations described by the skill are exposed independently through the CLI
and the `task_*` MCP tools.
