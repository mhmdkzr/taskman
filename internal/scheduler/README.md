# `scheduler`

Durable one-time and recurring message delivery on the NATS bus.

## What it provides

- Rows live in SQLite; per-message timers give low-latency delivery and a poll backstop replays or expires anything the timers missed.
- Delivery is exactly-once: each publish carries the row's id as its dedup id, so a crash between publish and record is deduplicated by the stream.
- Recurrences are fixed-interval schedules; each occurrence is its own `scheduler` row linked by `recurrence_id`.

## Invocation

`New(db, pub, grace)` builds the scheduler; `Schedule` / `ScheduleRecurring` / `Cancel` manage messages; `Run` serves it until the context is cancelled. The server starts it on boot; `Poll` can be called on demand.