# `store`

SQLite persistence for agent sessions and the scheduler.

## What it provides

- Two handles per database: RW for writes and RO for reads (opened with `mode=ro`, enforced by the storage engine).
- Sessions as a shared message chain: `sessions` + `options` + `forks` + `messages`, with fork-by-reference sessions.
- `Migrate` applies the embedded schema from `migrations/schema.sql` (idempotent, no versioning).
- Durable scheduler rows and recurrences.

## Invocation

`store.Open(path)` opens a Store; `store.Migrate` runs the schema. The server opens and migrates on startup. IDs are UUIDv7 everywhere; the embedded timestamp becomes `created_at`.

`SaveRun`/`AppendRun` persist one turn atomically; `GetSessionTranscript` reconstructs a session; `ForkSession` forks by reference. The RO handle backs read paths such as `subagent_result`.