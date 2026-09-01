package stats

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/clistat"

	"github.com/mhmdkzr/taskman/internal/app"
)

func assertNonNegativeUsed(t *testing.T, name string, r *clistat.Result) {
	t.Helper()
	if r == nil {
		t.Fatalf("%s result is nil", name)
		return
	}
	if r.Used < 0 {
		t.Fatalf("%s used must be >= 0, got %f", name, r.Used)
	}
}

func assertNonNegativeUsedAndPositiveTotal(t *testing.T, name string, r *clistat.Result) {
	t.Helper()
	assertNonNegativeUsed(t, name, r)
	if r.Total == nil {
		t.Fatalf("%s total is nil", name)
	}
	if *r.Total <= 0 {
		t.Fatalf("%s total must be > 0, got %f", name, *r.Total)
	}
	if r.Used > *r.Total {
		t.Fatalf("%s used exceeds total: used=%f total=%f", name, r.Used, *r.Total)
	}
}

func TestRead_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping smoke test in short mode")
	}

	s, err := Read()
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if s.Timestamp == "" {
		t.Fatal("expected non-empty timestamp")
	}
	if s.HostCPU == nil {
		t.Fatal("expected host CPU stats")
	}
	if s.HostMemory == nil {
		t.Fatal("expected host memory stats")
	}
	if s.Disk == nil {
		t.Fatal("expected disk stats")
	}
	assertNonNegativeUsed(t, "host_cpu", s.HostCPU)
	assertNonNegativeUsedAndPositiveTotal(t, "host_memory", s.HostMemory)
	assertNonNegativeUsedAndPositiveTotal(t, "disk", s.Disk)

	if s.IsContainerized {
		if s.ContainerCPU == nil {
			t.Fatal("expected container CPU stats when containerized")
		}
		if s.ContainerMemory == nil {
			t.Fatal("expected container memory stats when containerized")
		}
		assertNonNegativeUsed(t, "container_cpu", s.ContainerCPU)
		assertNonNegativeUsedAndPositiveTotal(t, "container_memory", s.ContainerMemory)
	}
}

func TestHandler_ReturnsStats(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var s Stats
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if s.Timestamp == "" {
		t.Fatal("expected non-empty timestamp")
	}
	if s.HostCPU == nil {
		t.Fatal("expected host CPU stats")
	}
	if s.HostMemory == nil {
		t.Fatal("expected host memory stats")
	}
	if s.Disk == nil {
		t.Fatal("expected disk stats")
	}
}

func TestHandler_ReadErrorReturns500(t *testing.T) {
	orig := readStats
	defer func() { readStats = orig }()
	readStats = func() (Stats, error) {
		return Stats{}, errors.New("boom")
	}

	handler := Handler()
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if errResp.Error != "boom" {
		t.Fatalf("error body: got %q want %q", errResp.Error, "boom")
	}
}

func TestRegisterRoutes_RegistersStatsEndpoint(t *testing.T) {
	a := app.App{Mux: http.NewServeMux()}
	RegisterRoutes(a)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()
	a.Mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_ContentType(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
}
