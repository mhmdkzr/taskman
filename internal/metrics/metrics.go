// Package metrics exposes Prometheus metrics for the Go application.
package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds the Prometheus registry and instruments HTTP requests.
type Metrics struct {
	registry            *prometheus.Registry
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
}

// New creates a Metrics instance with a dedicated registry and registers
// Go and process collectors as documented at https://prometheus.io/docs/guides/go-application/.
func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"method", "path", "code"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Histogram of HTTP request durations.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "code"},
		),
	}
	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.httpRequestsTotal,
		m.httpRequestDuration,
	)
	return m
}

// Registry returns the Prometheus registry used by the application.
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// Handler returns an HTTP handler that serves the Prometheus metrics exposition format
// at /metrics (as documented at https://prometheus.io/docs/guides/go-application/).
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware instruments HTTP requests with Prometheus metrics.
// It records http_requests_total and http_request_duration_seconds for each request.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		duration := time.Since(start).Seconds()
		code := strconv.Itoa(rec.status)
		path := r.URL.Path
		labels := prometheus.Labels{"method": r.Method, "path": path, "code": code}
		m.httpRequestsTotal.With(labels).Inc()
		m.httpRequestDuration.With(labels).Observe(duration)
	})
}

type statusRecorder struct {
	http.ResponseWriter

	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	if err != nil {
		return n, fmt.Errorf("write response: %w", err)
	}
	return n, nil
}
