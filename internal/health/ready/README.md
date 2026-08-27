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
    "database": {
      "status": "ok"
    },
    "nats": {
      "status": "ok"
    },
    "temporal": {
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
    "database": {
      "status": "ok"
    },
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
| `database` | `sql.DB.PingContext`                |
| `nats`     | `nats.Conn.Status() == CONNECTED`   |
| `temporal` | `temporalclient.Client.CheckHealth` |

## Example

```bash
curl -sS http://127.0.0.1:8090/ready   # via Caddy; direct app is :8080
```

```json
{
  "status": "ok",
  "checks": {
    "database": {
      "status": "ok"
    },
    "nats": {
      "status": "ok"
    },
    "temporal": {
      "status": "ok"
    }
  }
}
```
