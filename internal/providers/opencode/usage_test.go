package opencode

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/config"
)

const upstreamUsageBody = `{"usage":{` +
	`"rolling":{"status":"ok","percent":2,"resetsAt":"2026-09-07T12:12:30.464Z"},` +
	`"weekly":{"status":"ok","percent":0,"resetsAt":"2026-09-14T00:00:00.464Z"},` +
	`"monthly":{"status":"rate-limited","percent":100,"resetsAt":"2026-09-30T17:15:56.464Z"}}}`

const upstreamUnauthorizedBody = `{"type":"error","error":{"type":"AuthError","message":"Unauthorized"}}`

const upstreamNoSubscriptionBody = `{"type":"error","error":{"type":"EntitlementError","message":"OpenCode Go subscription required."}}`

// newUpstream starts a fake OpenCode usage endpoint serving one canned
// response and verifies the request path and authorization the client sends.
func newUpstream(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usage" {
			t.Errorf("upstream path = %q, want /usage", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("upstream authorization = %q, want %q", auth, "Bearer test-key")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func testConfig(baseURL string) config.ProviderConfig {
	return config.ProviderConfig{
		BaseURL:         baseURL,
		APIKeyOpenCode:  "test-key",
		Model:           "test-model",
		ReasoningEffort: "medium",
	}
}

func TestFetchUsage(t *testing.T) {
	server := newUpstream(t, http.StatusOK, upstreamUsageBody)
	got, err := FetchUsage(context.Background(), testConfig(server.URL))
	if err != nil {
		t.Fatalf("FetchUsage() error = %v", err)
	}

	tests := []struct {
		name    string
		window  Window
		status  WindowStatus
		percent int
		resets  time.Time
	}{
		{
			name:    "rolling",
			window:  got.Rolling,
			status:  StatusOK,
			percent: 2,
			resets:  time.Date(2026, 9, 7, 12, 12, 30, 464000000, time.UTC),
		},
		{
			name:    "weekly",
			window:  got.Weekly,
			status:  StatusOK,
			percent: 0,
			resets:  time.Date(2026, 9, 14, 0, 0, 0, 464000000, time.UTC),
		},
		{
			name:    "monthly",
			window:  got.Monthly,
			status:  StatusRateLimited,
			percent: 100,
			resets:  time.Date(2026, 9, 30, 17, 15, 56, 464000000, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.window.Status != test.status {
				t.Errorf("status = %q, want %q", test.window.Status, test.status)
			}
			if test.window.Percent != test.percent {
				t.Errorf("percent = %d, want %d", test.window.Percent, test.percent)
			}
			if !test.window.ResetsAt.Equal(test.resets) {
				t.Errorf("resets_at = %v, want %v", test.window.ResetsAt, test.resets)
			}
		})
	}
}

func TestFetchUsageUpstreamErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantErr    error
		wantSubstr string
	}{
		{
			name:    "unauthorized",
			status:  http.StatusUnauthorized,
			body:    upstreamUnauthorizedBody,
			wantErr: ErrUnauthorized,
		},
		{
			name:    "no subscription",
			status:  http.StatusForbidden,
			body:    upstreamNoSubscriptionBody,
			wantErr: ErrNoSubscription,
		},
		{
			name:       "unexpected status",
			status:     http.StatusInternalServerError,
			body:       "oops",
			wantSubstr: "HTTP 500",
		},
		{
			name:       "invalid json",
			status:     http.StatusOK,
			body:       "not-json",
			wantSubstr: "decode usage",
		},
		{
			name:       "invalid resets_at",
			status:     http.StatusOK,
			body:       `{"usage":{"rolling":{"status":"ok","percent":1,"resetsAt":"nope"}}}`,
			wantSubstr: "resets_at",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newUpstream(t, test.status, test.body)
			_, err := FetchUsage(context.Background(), testConfig(server.URL))
			if err == nil {
				t.Fatal("FetchUsage() error = nil, want error")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			if test.wantSubstr != "" && !strings.Contains(err.Error(), test.wantSubstr) {
				t.Fatalf("error = %q, want containing %q", err, test.wantSubstr)
			}
		})
	}
}

func TestFetchUsageUnconfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.ProviderConfig
	}{
		{
			name: "no base url",
			cfg:  config.ProviderConfig{APIKeyOpenCode: "test-key", Model: "m", ReasoningEffort: "medium"},
		},
		{
			name: "no api key",
			cfg:  config.ProviderConfig{BaseURL: "http://localhost", Model: "m", ReasoningEffort: "medium"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := FetchUsage(context.Background(), test.cfg)
			if !errors.Is(err, ErrNotConfigured) {
				t.Fatalf("error = %v, want %v", err, ErrNotConfigured)
			}
		})
	}
}

func TestUsageHandler(t *testing.T) {
	tests := []struct {
		name           string
		upstreamStatus int
		upstreamBody   string
		wantStatus     int
		wantBody       string
	}{
		{
			name:           "ok",
			upstreamStatus: http.StatusOK,
			upstreamBody:   upstreamUsageBody,
			wantStatus:     http.StatusOK,
			wantBody: `{"rolling":{"status":"ok","percent":2,"resets_at":"2026-09-07T12:12:30.464Z"},` +
				`"weekly":{"status":"ok","percent":0,"resets_at":"2026-09-14T00:00:00.464Z"},` +
				`"monthly":{"status":"rate-limited","percent":100,"resets_at":"2026-09-30T17:15:56.464Z"}}`,
		},
		{
			name:           "unauthorized",
			upstreamStatus: http.StatusUnauthorized,
			upstreamBody:   upstreamUnauthorizedBody,
			wantStatus:     http.StatusBadGateway,
			wantBody:       `{"error":"opencode: API key is missing or rejected: Unauthorized"}`,
		},
		{
			name:           "no subscription",
			upstreamStatus: http.StatusForbidden,
			upstreamBody:   upstreamNoSubscriptionBody,
			wantStatus:     http.StatusBadGateway,
			wantBody:       `{"error":"opencode: OpenCode Go subscription required: OpenCode Go subscription required."}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newUpstream(t, test.upstreamStatus, test.upstreamBody)
			mux := http.NewServeMux()
			RegisterRoutes(app.App{Cfg: config.Config{Provider: testConfig(server.URL)}, Mux: mux})

			request := httptest.NewRequest(http.MethodGet, "/providers/opencode/usage", nil)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d, body %s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			if got := strings.TrimSpace(recorder.Body.String()); got != test.wantBody {
				t.Errorf("body = %s, want %s", got, test.wantBody)
			}
		})
	}
}
