// Package app contains the embedded frontend dist and common app-level types.
package app

import "embed"

//go:embed frontend/dist
var FS embed.FS
