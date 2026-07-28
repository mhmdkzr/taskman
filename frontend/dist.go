// Package app contains the embedded frontend dist and common app-level types.
package frontend

import "embed"

//go:embed dist
var FS embed.FS
