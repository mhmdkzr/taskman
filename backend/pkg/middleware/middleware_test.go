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
