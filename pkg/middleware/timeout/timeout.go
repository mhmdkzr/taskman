package timeout

import (
	"context"
	"net/http"
	"time"

	"github.com/mhmdkzr/app/pkg/middleware"
)

// New returns a middleware that applies a timeout to requests.
func New(d time.Duration) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
