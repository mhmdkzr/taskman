# `pkg/stats`

System resource monitoring. `Read()` collects host/container CPU, memory, and disk usage via `clistat`. `Handler()` returns an `http.HandlerFunc` that serves stats as JSON via a `GET /stats` endpoint.
