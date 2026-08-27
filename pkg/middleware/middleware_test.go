package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChain_ExecutesAllInOrder(t *testing.T) {
	var order []int

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 1)
			next.ServeHTTP(w, r)
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 2)
			next.ServeHTTP(w, r)
		})
	}
	mw3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 3)
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, 4)
	})

	chained := Chain(handler, mw1, mw2, mw3)
	chained.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []int{1, 2, 3, 4}
	if len(order) != len(want) {
		t.Fatalf("execution order = %v, want %v", order, want)
	}
	for i, v := range order {
		if v != want[i] {
			t.Fatalf("execution order[%d] = %d, want %d", i, v, want[i])
		}
	}
}

func TestChain_EmptyMiddlewares(t *testing.T) {
	var called bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	chained := Chain(handler)
	chained.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !called {
		t.Fatal("handler should be called")
	}
}

func TestChain_ReplacesContext(t *testing.T) {
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithRequestID(r.Context(), "test-id")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	var gotID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = RequestIDFromContext(r.Context())
	})

	Chain(handler, mw).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if gotID != "test-id" {
		t.Fatalf("request id = %q, want %q", gotID, "test-id")
	}
}

func TestSkip(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantWrapped  bool
		wantResponse string
	}{
		{name: "matching request bypasses middleware", path: "/health", wantResponse: "next"},
		{name: "other request uses middleware", path: "/api", wantWrapped: true, wantResponse: "middleware"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wrapped bool
			mw := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					wrapped = true
					w.WriteHeader(http.StatusAccepted)
					_, _ = w.Write([]byte("middleware"))
				})
			}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("next"))
			})

			rec := httptest.NewRecorder()
			Skip(mw, func(r *http.Request) bool { return r.URL.Path == "/health" })(handler).
				ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if wrapped != tt.wantWrapped {
				t.Fatalf("middleware called = %t, want %t", wrapped, tt.wantWrapped)
			}
			if rec.Body.String() != tt.wantResponse {
				t.Fatalf("response body = %q, want %q", rec.Body.String(), tt.wantResponse)
			}
		})
	}
}
