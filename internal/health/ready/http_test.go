package ready

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/routes"
)

func TestHandle_AllNilDeps_ReturnsDegraded(t *testing.T) {
	a := app.App{Mux: http.NewServeMux()}
	routes.RegisterRoutes(a, NewHandler(a).Route())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	a.Mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status mismatch: got=%d want=%d", rec.Code, http.StatusServiceUnavailable)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "degraded" {
		t.Fatalf("status mismatch: got=%q want=%q", resp.Status, "degraded")
	}

	allFailing := []*CheckResult{
		resp.Checks.Database,
		resp.Checks.NATS,
		resp.Checks.JetStream,
		resp.Checks.Temporal,
		resp.Checks.TigerBeetle,
		resp.Checks.Zitadel,
		resp.Checks.Mailer,
	}
	for i, r := range allFailing {
		if r == nil {
			t.Fatalf("nil result for required check at index %d", i)
		}
		if r.Status != "error" {
			t.Fatalf("check status: got=%q want=%q", r.Status, "error")
		}
	}
}

func TestHandle_ReturnsJSON(t *testing.T) {
	a := app.App{Mux: http.NewServeMux()}
	routes.RegisterRoutes(a, NewHandler(a).Route())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	a.Mux.ServeHTTP(rec, req)

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
