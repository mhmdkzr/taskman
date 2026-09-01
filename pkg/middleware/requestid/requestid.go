package requestid

import (
	"errors"
	"net/http"

	"github.com/mhmdkzr/taskman/pkg/jsonresp"
	"github.com/mhmdkzr/taskman/pkg/middleware"
)

var errMissingRequestID = errors.New("X-Request-ID header is required")

// Require rejects requests that do not provide an X-Request-ID header.
func Require() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Request-ID") == "" {
				jsonresp.WriteHTTPError(w, http.StatusBadRequest, errMissingRequestID)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
