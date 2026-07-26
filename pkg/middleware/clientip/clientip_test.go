package clientip

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mhmdkzr/app/pkg/middleware"
)

func extractIP(r *http.Request) string {
	var ip string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip = middleware.ClientIPFromContext(r.Context())
	})

	New("192.168.0.0/16")(handler).ServeHTTP(httptest.NewRecorder(), r)
	return ip
}

func TestClientIP_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.1")
	r.RemoteAddr = "192.168.1.1:12345"

	ip := extractIP(r)
	if ip != "203.0.113.1" {
		t.Fatalf("ip = %q, want %q", ip, "203.0.113.1")
	}
}

func TestClientIP_XForwardedForTakesFirst(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.2, 10.0.0.1")
	r.RemoteAddr = "192.168.1.1:12345"

	ip := extractIP(r)
	if ip != "203.0.113.1" {
		t.Fatalf("ip = %q, want %q", ip, "203.0.113.1")
	}
}

func TestClientIP_XRealIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Real-IP", "198.51.100.2")
	r.RemoteAddr = "192.168.1.1:12345"

	ip := extractIP(r)
	if ip != "198.51.100.2" {
		t.Fatalf("ip = %q, want %q", ip, "198.51.100.2")
	}
}

func TestClientIP_XForwardedForPreferredOverXRealIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.1")
	r.Header.Set("X-Real-IP", "198.51.100.2")
	r.RemoteAddr = "192.168.1.1:12345"

	ip := extractIP(r)
	if ip != "203.0.113.1" {
		t.Fatalf("ip = %q, want %q", ip, "203.0.113.1")
	}
}

func TestClientIP_FallbackToRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.5:54321"

	ip := extractIP(r)
	if ip != "10.0.0.5" {
		t.Fatalf("ip = %q, want %q", ip, "10.0.0.5")
	}
}

func TestClientIP_RemoteAddrWithoutPort(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.5"

	ip := extractIP(r)
	if ip != "10.0.0.5" {
		t.Fatalf("ip = %q, want %q", ip, "10.0.0.5")
	}
}

func TestClientIP_EmptyHeadersUsesRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "")
	r.RemoteAddr = "10.0.0.5:12345"

	ip := extractIP(r)
	if ip != "10.0.0.5" {
		t.Fatalf("ip = %q, want %q", ip, "10.0.0.5")
	}
}

func TestNew_CheckType(t *testing.T) {
	mw := New()
	if mw == nil {
		t.Fatal("New() should not return nil")
	}
}

func TestExtract_IPv6(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "[::1]:12345"

	ip := extractIP(r)
	if ip != "::1" {
		t.Fatalf("ip = %q, want %q", ip, "::1")
	}
}

func TestClientIP_UntrustedSourceUsesRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.1")
	r.RemoteAddr = "203.0.113.99:12345"

	ip := extractIP(r)
	if ip != "203.0.113.99" {
		t.Fatalf("ip = %q, want %q", ip, "203.0.113.99")
	}
}

func TestClientIP_DefaultTrustsPrivateProxies(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.1")
	r.RemoteAddr = "192.168.1.1:12345"

	var ip string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip = middleware.ClientIPFromContext(r.Context())
	})
	New()(handler).ServeHTTP(httptest.NewRecorder(), r)

	if ip != "203.0.113.1" {
		t.Fatalf("ip = %q, want %q", ip, "203.0.113.1")
	}
}

func TestClientIP_PublicSourceIgnoresHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.1")
	r.RemoteAddr = "203.0.113.99:12345"

	var ip string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip = middleware.ClientIPFromContext(r.Context())
	})
	New()(handler).ServeHTTP(httptest.NewRecorder(), r)

	if ip != "203.0.113.99" {
		t.Fatalf("ip = %q, want %q", ip, "203.0.113.99")
	}
}
