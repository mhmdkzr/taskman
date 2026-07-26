package get

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mhmdkzr/app/internal/app"
)

func TestHandle_ReturnsOK(t *testing.T) {
	a := app.App{Mux: http.NewServeMux()}
	a.RegisterRoutes(NewHandler(a).Route())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	a.Mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch: got=%d want=%d", rec.Code, http.StatusOK)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status mismatch: got=%q want=%q", resp.Status, "ok")
	}
}
