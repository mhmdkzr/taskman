# `register`

Registration hub — the single call site that wires up the entire system by delegating to each module's register slice.

## Registration Functions

| Function | What it registers |
|---|---|
| `RegisterRoutes(a app.App)` | All HTTP routes |
| `RegisterActivities(w worker.Worker, a app.App)` | All Temporal activities |
| `RegisterWorkflows(w worker.Worker, a app.App)` | All Temporal workflows |
| `RegisterEvents(w worker.Worker, a app.App)` | Event-driven handlers (currently no-op) |

## Delegation Chain

```
RegisterRoutes
  ├── health/get.RegisterRoutes
  └── stats.RegisterRoutes

RegisterActivities (currently empty)

RegisterWorkflows (currently empty)

RegisterEvents (currently empty)
```
