package middleware

import (
	"net/http"
	"slices"
)

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares to a handler in the order they are provided.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for _, v := range slices.Backward(mws) {
		h = v(h)
	}
	return h
}

// Skip wraps mw so that it is bypassed for requests matched by skip.
func Skip(mw Middleware, skip func(*http.Request) bool) Middleware {
	return func(next http.Handler) http.Handler {
		wrapped := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip(r) {
				next.ServeHTTP(w, r)
				return
			}
			wrapped.ServeHTTP(w, r)
		})
	}
}
