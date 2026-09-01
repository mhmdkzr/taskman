# `tools`

The tool set an agent can call.

## What it provides

- `bash` — run shell commands.
- `files` — grep, glob, read, write, edit, diff, append.
- `telegram` — send messages to and read the configured Telegram channel (`telegram_send` / `telegram_read`), gated on credentials.
- `spawn` — `spawn_subagent` / `subagent_result` for nested sub-agents, backed by a process-wide `Runner`.
- `schedule` — the `agent.run` wire contract (`RunRequest`, `RunSubject`, `MeaningfulOverride`) plus the `Schedule` / `ScheduleRecurring` helpers. The scheduling *tools* are intentionally not included; the durable scheduler runtime lives in `internal/scheduler`.

`tools.Tools(tg)` assembles bash + files + telegram. The spawn tools are appended by `agent.DefaultTools`.