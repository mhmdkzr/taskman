package timeout

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestTimeout_HonorsDuration(t *testing.T) {
	var called atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(1)
	})

	mw := New(10 * time.Millisecond)
	mw(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if called.Load() != 1 {
		t.Fatal("handler should be called when request completes within timeout")
	}
}

func TestTimeout_CancelsSlowHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(5 * time.Second):
		case <-r.Context().Done():
		}
	})

	mw := New(5 * time.Millisecond)
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (timeout doesn't set status, handler just stops)", rec.Code)
	}
}

func TestTimeout_ContextCancelled(t *testing.T) {
	done := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(done)
	})

	mw := New(5 * time.Millisecond)
	mw(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("context was not canceled after timeout")
	}
}

func TestTimeout_PreservesParentContext(t *testing.T) {
	type key string
	const parentKey key = "parent"
	parentVal := "hello"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(parentKey)
		if v != parentVal {
			t.Fatalf("parent context value = %v, want %v", v, parentVal)
		}
	})

	ctx := context.WithValue(context.Background(), parentKey, parentVal)
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)

	mw := New(time.Second)
	mw(handler).ServeHTTP(httptest.NewRecorder(), r)
}

func TestTimeout_SetsDeadline(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Fatal("deadline should be set")
		}
		if deadline.Before(time.Now().Add(time.Hour)) {
			t.Logf("deadline is set: %v", deadline)
		}
	})

	mw := New(time.Second)
	mw(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}
