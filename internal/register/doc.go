// Package register is the root-level aggregation hub. It delegates registration of HTTP routes
// to all module-level register packages following the three-layer delegation pattern (slice -> module -> root).
package register
