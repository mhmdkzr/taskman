# `pkg/stats`

System resource monitoring for the service. Collects host and container
CPU, memory, and disk usage via `coder/clistat` and exposes it over HTTP.

## What it provides

- `Read() (Stats, error)` collects current usage into a `Stats` value with:
  - `Timestamp`
  - `HostCPU`, `HostMemory` (GiB), `Disk` (root filesystem, GiB)
  - `IsContainerized`
  - `ContainerCPU`, `ContainerMemory` — populated only when running inside a container
- `Handler() http.HandlerFunc` serves the latest `Stats` as JSON.
- `RegisterRoutes(a app.App)` wires the handler to `GET /stats`.

## Behavior

- Any failure while reading host stats is returned as an error; the HTTP
  handler maps it to `500`.
- Container metrics require a containerized environment; otherwise those
  fields are omitted from the JSON output (`omitempty`).

## Usage

`stats.RegisterRoutes(a)` wires the handler to `GET /stats`:

```bash
curl http://127.0.0.1:8080/stats
```

Each `clistat.Result` serializes as `{"total": ..., "unit": "...", "used": ...}`;
`total` is `null` for non-finite resources such as CPU (unit `cores`). Memory
and disk use GiB prefixes; disk usage covers the root filesystem:

```json
{
  "timestamp": "2026-08-22 10:00:00.123 +0000 UTC",
  "host_cpu": {
    "total": null,
    "unit": "cores",
    "used": 0.4
  },
  "host_memory": {
    "total": 16,
    "unit": "GiB",
    "used": 4
  },
  "disk": {
    "total": 320,
    "unit": "GiB",
    "used": 100
  },
  "is_containerized": false
}
```
