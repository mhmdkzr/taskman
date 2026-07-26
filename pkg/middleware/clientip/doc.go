// Package clientip provides HTTP middleware that extracts the real client IP address from
// requests, respecting X-Forwarded-For and X-Real-IP headers for trusted proxy CIDR ranges.
package clientip
