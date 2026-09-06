// Package tool provides read-only lookup tools for registered tool
// definitions, in its get and list subpackages.
package tool

import "uuid"

// Tool is one tools row: a registered tool's name, description, and schemas.
type Tool struct {
	ID           uuid.UUID
	Name         string
	Description  string
	InputSchema  string
	OutputSchema string
}
