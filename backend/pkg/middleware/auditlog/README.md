# `pkg/middleware/auditlog`

Captures HTTP request/response metadata, publishes audit events to JetStream, and persists them to PostgreSQL.

## Components

- **Middleware** (`auditlog.go`): Intercepts HTTP requests, captures request/response metadata, and publishes `Event` messages to the `api.audit` subject on the `API_AUDIT` JetStream stream. Supports optional redaction via `NewWithRedactor`.
- **Consumer** (`consumer.go`): Durable JetStream consumer named `audit-log-db-writer` that reads events from `api.audit` and writes them to `core.audit_logs`.
- **Repository** (`repo.go`): Database access that inserts audit events into `core.audit_logs` with idempotent deduplication by `request_id`.

## Behavior

Persistence is idempotent: `request_id` is the primary key and duplicate deliveries are ignored with `ON CONFLICT DO NOTHING`. Request and response bodies are stored as published.

## Invocation

The middleware is wired in `internal/process.Start`:

```go
auditlog.NewWithRedactor(a.Deps.JS, redact.AuditEvent)
```

The consumer starts in the same startup function after PostgreSQL, NATS JetStream, and application dependencies are initialized.

Manual publish example:

```bash
nats pub api.audit '{"request_id":"example","timestamp":"2026-06-13T00:00:00Z","duration":1000000,"request":{"id":"example","method":"GET","path":"/health","query":"","remote_addr":"127.0.0.1:1","user_agent":"curl","proto":"HTTP/1.1","headers":{}},"response":{"status_code":200,"headers":{},"size":0}}'
```

## Storage

Events are stored in `core.audit_logs` with request metadata, response metadata, headers as `jsonb`, and bodies as `bytea`.
