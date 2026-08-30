# Metrics

Exposes Prometheus metrics for the Go application via `GET /metrics` (see https://prometheus.io/docs/guides/go-application/).

## What it provides

- Go runtime metrics (`go_*`) and process metrics (`process_*`) via `collectors.NewGoCollector` and `collectors.NewProcessCollector`.
- HTTP instrumentation: `http_requests_total` (counter, labels `method`, `path`, `code`) and `http_request_duration_seconds` (histogram, same labels) recorded by `metrics.Middleware`.
- Registry: a dedicated `prometheus.Registry` (not the global default) is used; the handler is `promhttp.HandlerFor(registry, ...)`.

## How it is invoked

- **HTTP**: `GET /metrics` → Prometheus exposition format (text/plain). Scraped by Prometheus (`prom/prometheus`) every 15s via `config/prometheus/prometheus.yml` job `app`.
- **Middleware**: `metrics.Middleware` is chained in `internal/process/start.go` around the application mux to instrument all HTTP requests.

## Curl example

```bash
curl http://localhost:8080/metrics
# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.
# TYPE go_gc_duration_seconds summary
# HELP http_requests_total Total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{code="200",method="GET",path="/health"} 1
```

## Prometheus config

```yaml
- job_name: 'app'
  metrics_path: /metrics
  static_configs:
    - targets: ['app:8080']
```

## Temporal SDK metrics

Temporal SDK metrics are separate (Tally/OpenTelemetry) and configured via `PROMETHEUS_ENDPOINT` on the Temporal service.
