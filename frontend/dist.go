// Package frontend embeds the frontend dist directory.
package frontend

import "embed"

//go:embed dist
var FS embed.FS
