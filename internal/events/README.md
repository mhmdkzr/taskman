# `events`

Every message type published on the agent bus.

## What it provides

Each event implements `Subject()` (dot-separated, under `agent.>` or `scheduler.>`) and `MsgID()` (a deterministic content hash via `eventID`), so re-publishing the same event always deduplicates. New event types must implement both.

Event groups: turn lifecycle, model request/response, steps, tool calls, generation finish, panics, sub-agents, scheduled runs (`agent.run.*`), and scheduler notices (`scheduler.expired`, `scheduler.recurring.missed`).