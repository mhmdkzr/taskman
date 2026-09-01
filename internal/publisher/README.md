# `publisher`

The sole gateway to the NATS bus: everything that sends or consumes messages goes through it, never the raw nats/JetStream APIs.

## What it provides

- `Publish` / `PublishMsg` — send a message with a dedup id in the `Nats-Msg-Id` header for exactly-once delivery.
- `Subscribe` — durable JetStream consumer filtered to a subject.
- `CreateStreams` — creates the `SCION` stream (`agent.>`, `scheduler.>`) with a 24h dedup window; registered from `internal/streams`.
- `ConnectOn` (embedded server, tests only) / `ConnectURL` (production).

## Notes

`StreamName` (`SCION`) and the dedup window are defined here. `Event` is any value with `Subject()` and `MsgID()`; event definitions live in `internal/events`.