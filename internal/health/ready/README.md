# `health/ready`

Readiness check slice that probes all external dependencies and reports their status.

## Route

- `GET /ready`

## Behavior

- Probes all external dependencies with a 5-second per-check timeout.
- Returns `200 OK` when all checks pass.
- Returns `503 Service Unavailable` when any check fails, with per-dependency details.
- Does **not** block the startup sequence; it is a runtime endpoint for load balancers and orchestrators.

## Response format

```json
{
  "status": "ok",
  "checks": {
    "nats": {
      "status": "ok"
    },
    "jetstream": {
      "status": "ok"
    }
  }
}
```

A failing check includes an `error` field:

```json
{
  "status": "degraded",
  "checks": {
    "nats": {
      "status": "error",
      "error": "not connected"
    }
  }
}
```

## Dependencies checked

| Check key  | What is probed                      |
|------------|-------------------------------------|
| `nats`     | `nats.Conn.Status() == CONNECTED`   |
| `jetstream` | JetStream `AccountInfo`             |

## Example

```bash
curl -sS http://127.0.0.1:8080/ready
```
