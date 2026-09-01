# `register`

Registration hub — the single call site that wires up the entire system by delegating to each module's register slice.

## Registration Functions

| Function | What it registers |
|---|---|
| `RegisterRoutes(a app.App)` | All HTTP routes |

## Delegation Chain

```
RegisterRoutes
  └── health/register.RegisterRoutes
        ├── health/get.RegisterRoutes
        └── health/ready.RegisterRoutes
```
