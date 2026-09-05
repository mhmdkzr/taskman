# `register`

Registration hub — the single call site that wires up the entire system.

## Registration Functions

| Function | What it registers |
|---|---|
| `RegisterRoutes(a app.App, m *metrics.Metrics)` | All HTTP routes |
| `RegisterActivities()` | All Temporal activities (currently empty) |
| `RegisterWorkflows()` | All Temporal workflows (currently empty) |
| `RegisterEvents()` | Event-driven handlers (currently empty) |
