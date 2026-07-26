package pagination

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Meta contains pagination metadata for list responses.
type Meta struct {
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total,omitempty"`
}

const (
	// DefaultPage is the default page number.
	DefaultPage = 1
	// DefaultSize is the default page size.
	DefaultSize = 10
	// MaxPageSize is the maximum allowed page size.
	MaxPageSize = 1000
)

// Normalize sets default page and size values on the metadata.
func Normalize(m Meta) Meta {
	if m.Page == 0 {
		m.Page = DefaultPage
	}
	if m.Size == 0 {
		m.Size = DefaultSize
	}
	return m
}

// Validate checks that the pagination parameters are within valid bounds.
func Validate(m Meta) error {
	if m.Page < 1 {
		return fmt.Errorf("page must be greater than 0")
	}
	if m.Size < 1 {
		return fmt.Errorf("size must be greater than 0")
	}
	if m.Size > MaxPageSize {
		return fmt.Errorf("size must not exceed %d", MaxPageSize)
	}
	return nil
}

// ParseRequest extracts pagination metadata from HTTP request query parameters.
// Returns nil, nil if both page and size are empty.
//
//nolint:nilnil // Missing pagination query parameters are represented by a nil Meta and nil error.
func ParseRequest(r *http.Request) (*Meta, error) {
	rawPage := strings.TrimSpace(r.URL.Query().Get("page"))
	rawSize := strings.TrimSpace(r.URL.Query().Get("size"))
	if rawPage == "" && rawSize == "" {
		return nil, nil
	}

	var m Meta
	if rawPage != "" {
		page, err := strconv.Atoi(rawPage)
		if err != nil {
			return nil, fmt.Errorf("invalid page: %w", err)
		}
		m.Page = page
	}
	if rawSize != "" {
		size, err := strconv.Atoi(rawSize)
		if err != nil {
			return nil, fmt.Errorf("invalid size: %w", err)
		}
		m.Size = size
	}
	return &m, nil
}
