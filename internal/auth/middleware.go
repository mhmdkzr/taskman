package auth

import (
	"net/http"

	"github.com/mhmdkzr/app/pkg/middleware"
)

const userIDHeader = "X-User-ID"

func New() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get(userIDHeader)
			if userID != "" {
				ctx := ContextWithUser(r.Context(), UserInfo{UserID: userID})
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}
