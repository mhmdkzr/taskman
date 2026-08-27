# `auditlog`

Import path: `github.com/mhmdkzr/app/pkg/middleware/auditlog`

Captures HTTP request/response metadata and publishes audit events to JetStream.

## Components

- **Middleware** (`auditlog.go`): Intercepts HTTP requests, captures request/response metadata, and publishes events to the configured subject. Supports optional redaction via `NewWithRedactor`.
- **Stream** (`streams.go`): Creates or updates the configured JetStream stream and binds it to the configured subject.

## Invocation

Define service-specific names once and use the same configuration for the stream and middleware:

```go
cfg := auditlog.Config{
    Subject: "network.api.audit",
    Stream:  "NETWORK_API_AUDIT",
}
if err := auditlog.CreateStream(ctx, js, cfg); err != nil {
    return err
}
mw, err := auditlog.NewWithRedactor(js, cfg, redact.AuditEvent)
if err != nil {
    return err
}
```

Manual publish example:

```bash
nats pub network.api.audit '{"request_id":"example","timestamp":"2026-06-13T00:00:00Z","duration":1000000,"request":{"id":"example","method":"GET","path":"/health","query":"","remote_addr":"127.0.0.1:1","user_agent":"curl","proto":"HTTP/1.1","headers":{}},"response":{"status_code":200,"headers":{},"size":0}}'
```
