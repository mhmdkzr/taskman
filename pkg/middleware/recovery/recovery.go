package recovery

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mhmdkzr/loop/pkg/jsonresp"
	"github.com/mhmdkzr/loop/pkg/middleware"
)

var errInternalServerError = errors.New("internal server error")

// New returns a middleware that recovers from panics in downstream handlers, logs them
// with the request ID, and writes a 500 response instead of crashing the server.
func New() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer recoverAndRespond(w, r)
			next.ServeHTTP(w, r)
		})
	}
}

// recoverAndRespond recovers a panic from the current goroutine, logs it, and writes a
// 500 response. It is a no-op when there is nothing to recover.
func recoverAndRespond(w http.ResponseWriter, r *http.Request) {
	rec := recover()
	if rec == nil {
		return
	}

	slog.Error("panic in http handler",
		"panic", rec,
		"request_id", middleware.RequestIDFromContext(r.Context()),
		"method", r.Method,
		"path", r.URL.Path,
	)

	jsonresp.WriteHTTPError(w, http.StatusInternalServerError, errInternalServerError)
}
