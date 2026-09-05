package recovery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mhmdkzr/loop/pkg/middleware"
)

func TestRecovery_RecoversFromPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	mw := New()
	rec := httptest.NewRecorder()

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic escaped middleware: %v", r)
			}
		}()
		mw(handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	}()

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] == "" {
		t.Fatal("expected error message in response body")
	}
}

func TestRecovery_PassesThroughWithoutPanic(t *testing.T) {
	var called bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	mw := New()
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !called {
		t.Fatal("handler should be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRecovery_IncludesRequestIDInLogContext(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := middleware.RequestIDFromContext(r.Context()); id != "req-1" {
			t.Fatalf("request id = %q, want %q", id, "req-1")
		}
		panic("boom")
	})

	mw := New()
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(
		middleware.WithRequestID(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "req-1"),
	)
	mw(handler).ServeHTTP(rec, r)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
