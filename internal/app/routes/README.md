# `routes`

Central HTTP route registration for the app APIs. Public API slices declare
their endpoints as `Route` descriptors and hand them to `RegisterRoutes`, which
applies the configured `Server.BasePath` centrally and mounts each pattern on
the application mux.

## What it provides

- `Route` — a single HTTP route descriptor: `Method`, `Path`, and `Handler`.
- `RegisterRoutes(a app.App, routes ...Route)` — registers each route on the
  application mux, prepending the configured `Cfg.Server.BasePath` to every
  path. This is the registration entry point slices use.
- `Handle(a app.App, method, path string, handler http.HandlerFunc)` — the
  per-route worker behind `RegisterRoutes`: builds the `Method Path` pattern
  via `joinBasePath` and mounts it with `Mux.HandleFunc`.

## Behavior

- `Cfg.Server.BasePath` is applied centrally here, so route definitions stay
  module-relative and usually begin with `/`. A blank `BasePath` falls back to
  `/`.
- Route patterns use Go `net/http` method patterns; path parameters are read
  with `r.PathValue(...)`.
- Slices register exclusively via `Route` descriptors passed to
  `RegisterRoutes`.
