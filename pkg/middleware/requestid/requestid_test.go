package requestid

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequire(t *testing.T) {
	tests := []struct {
		name       string
		requestID  string
		wantStatus int
		wantCalled bool
	}{
		{name: "missing", wantStatus: http.StatusBadRequest},
		{name: "present", requestID: "req-1", wantStatus: http.StatusNoContent, wantCalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.requestID != "" {
				req.Header.Set("X-Request-ID", tt.requestID)
			}
			rec := httptest.NewRecorder()

			Require()(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Fatalf("next called = %t, want %t", called, tt.wantCalled)
			}
		})
	}
}
