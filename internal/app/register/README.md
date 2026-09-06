# `register`

Registration hub — the single call site that wires up the entire system.

## Registration Functions

| Function | What it registers |
|---|---|
| `RegisterRoutes(a app.App)` | All HTTP routes |
| `RegisterEvents()` | Event-driven handlers (currently empty) |
