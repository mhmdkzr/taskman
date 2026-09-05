# Sample: messy JSON blocks

A minified one-liner:

```json
{
  "status": "ok",
  "checks": {
    "database": {
      "status": "ok"
    }
  }
}
```

Hand-aligned padding:

```json
{
  "status": "degraded",
  "checks": {
    "nats": {
      "status": "error",
      "error": "not connected"
    },
    "database": {
      "status": "ok"
    }
  }
}
```

Non-JSON fences must be untouched:

```bash
curl -sS http://127.0.0.1:8090/ready
```

```
plain fence, no language
```

Text after the last block.
