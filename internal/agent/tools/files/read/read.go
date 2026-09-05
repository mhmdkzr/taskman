// Package read implements bounded file reading.
package read

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const (
	defaultOffset = 1
	defaultLimit  = 2000
	maxLineBytes  = 1 << 20 // 1MB per line
	initialBuffer = 64 << 10
)

var errPathRequired = errors.New("path is required")

type output struct {
	Content   string `json:"content"`
	LineCount int    `json:"line_count"`
	Truncated bool   `json:"truncated"`
}

func execute(_ context.Context, in input) (output, error) {
	offset := in.Offset
	if offset < 1 {
		offset = defaultOffset
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	file, err := os.Open(in.Path)
	if err != nil {
		return output{}, fmt.Errorf("read: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("close file after read", "path", in.Path, "error", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, initialBuffer), maxLineBytes)

	var lines []string
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount < offset {
			continue
		}
		if len(lines) < limit {
			lines = append(lines, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		return output{}, fmt.Errorf("read: %w", err)
	}

	truncated := offset+len(lines) <= lineCount
	return output{
		Content:   strings.Join(lines, "\n"),
		LineCount: lineCount,
		Truncated: truncated,
	}, nil
}
