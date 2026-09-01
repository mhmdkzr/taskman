# `protocol`

The wire contract between an agent client and the `internal/server` runtime: the request/reply envelope exchanged on `taskman.request` plus each command's payload.

## What it provides

- `Request` / `Reply` envelopes, `CommandRun` / `CommandPing`, and `RunRequest` / `RunReply` / `PingReply` payloads.
- `RunRequest.Validate` enforces coherence rules (prompt required, fork/session mutual exclusion, fork requires turn).

## Notes

This package is the one place client and server must agree, so it has no dependencies on the rest of the application. It is deliberately dependency-free.