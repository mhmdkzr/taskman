package logging

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/mhmdkzr/taskman/pkg/middleware"
)

const (
	serverErrorStatus = 500
	clientErrorStatus = 400
)

// sensitiveQueryKeys contains query parameter keys whose values should be redacted.
var sensitiveQueryKeys = map[string]bool{
	"token":     true,
	"api_key":   true,
	"apikey":    true,
	"secret":    true,
	"signature": true,
	"password":  true,
	"auth":      true,
}

// redactQuery replaces sensitive query parameter values with REDACTED.
func redactQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	vals, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}
	for k := range vals {
		if sensitiveQueryKeys[k] {
			vals[k] = []string{"REDACTED"}
		}
	}
	return vals.Encode()
}

// New returns a middleware that logs request completion details.
func New() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t0 := time.Now()
			next.ServeHTTP(w, r)

			level := slog.LevelInfo
			rec, ok := middleware.RecorderFromContext(r)
			if ok {
				if rec.StatusCode >= serverErrorStatus {
					level = slog.LevelError
				} else if rec.StatusCode >= clientErrorStatus {
					level = slog.LevelWarn
				}
			}

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", redactQuery(r.URL.RawQuery)),
				slog.Duration("duration", time.Since(t0)),
			}

			if id := middleware.RequestIDFromContext(r.Context()); id != "" {
				attrs = append(attrs, slog.String("request_id", id))
			}
			if ip := middleware.ClientIPFromContext(r.Context()); ip != "" {
				attrs = append(attrs, slog.String("ip", ip))
			}
			if ua := r.UserAgent(); ua != "" {
				attrs = append(attrs, slog.String("user_agent", ua))
			}
			if rec != nil {
				attrs = append(
					attrs,
					slog.Int("status", rec.StatusCode),
					slog.Int64("response_size", rec.BytesWritten),
				)
			}

			slog.LogAttrs(r.Context(), level, "request completed", attrs...)
		})
	}
}
