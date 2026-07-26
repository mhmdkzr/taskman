package clientip

import (
	"net"
	"net/http"
	"strings"

	"github.com/mhmdkzr/app/pkg/middleware"
)

// defaultTrustedCIDRs contains the default trusted CIDR ranges.
var defaultTrustedCIDRs = []string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"::1/128",
}

// New returns a middleware that extracts the client IP from the request.
func New(trustedCIDRs ...string) middleware.Middleware {
	if len(trustedCIDRs) == 0 {
		trustedCIDRs = defaultTrustedCIDRs
	}
	var trusted []*net.IPNet
	for _, c := range trustedCIDRs {
		_, n, err := net.ParseCIDR(c)
		if err == nil {
			trusted = append(trusted, n)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extract(r, trusted)
			ctx := middleware.WithClientIP(r.Context(), ip)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// isTrusted checks whether the given address is in any of the trusted CIDR ranges.
func isTrusted(addr string, trusted []*net.IPNet) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// extract extracts the client IP from the request, respecting trusted proxies.
func extract(r *http.Request, trusted []*net.IPNet) string {
	if isTrusted(r.RemoteAddr, trusted) {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			if i := strings.IndexByte(fwd, ','); i >= 0 {
				fwd = fwd[:i]
			}
			if ip := strings.TrimSpace(fwd); ip != "" {
				return ip
			}
		}

		if real := r.Header.Get("X-Real-IP"); real != "" {
			if ip := strings.TrimSpace(real); ip != "" {
				return ip
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
