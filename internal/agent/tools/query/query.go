// Package query provides safe, read-only SQL access to the agent database.
package query

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mhmdkzr/loop/internal/store"
)

const (
	queryTimeout   = 10 * time.Second
	maxRows        = 1000
	defaultCellLen = 200
	maxCellLen     = 10000
)

func execute(ctx context.Context, st *store.Store, in Input) (Output, error) {
	if st == nil {
		return Output{}, fmt.Errorf("database is required")
	}
	q := strings.TrimSpace(in.Query)
	cellLen := defaultCellLen
	if in.MaxCellLen != nil {
		cellLen = *in.MaxCellLen
	}
	if err := checkReadOnlyQuery(q); err != nil {
		return Output{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	rows, err := st.RO().QueryContext(ctx, q)
	if err != nil {
		return Output{}, fmt.Errorf("query: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.Error("close query rows", "error", closeErr)
		}
	}()
	out, err := formatRows(rows, cellLen)
	if err != nil {
		return Output{}, fmt.Errorf("query: %w", err)
	}
	return Output{Table: out}, nil
}

func checkReadOnlyQuery(q string) error {
	i := skipWhitespace(q, 0)
	start := i
	for i < len(q) && queryIdent(q[i]) {
		i++
	}
	if i == start {
		return fmt.Errorf("query: no statement found")
	}
	switch strings.ToUpper(q[start:i]) {
	case "SELECT", "WITH", "EXPLAIN":
	default:
		return fmt.Errorf("query: only read-only statements are allowed (SELECT, WITH, EXPLAIN)")
	}

	for i < len(q) {
		if i = skipWhitespace(q, i); i >= len(q) {
			break
		}
		switch q[i] {
		case '\'', '"', '`':
			i = skipQueryString(q, i)
		case ';':
			if j := skipWhitespace(q, i+1); j < len(q) {
				return fmt.Errorf("query: multiple statements are not allowed")
			}
			return nil
		default:
			i++
		}
	}
	return nil
}

func querySpace(b byte) bool {
	switch b {
	case ' ', '\t', '\r', '\n', '\v', '\f':
		return true
	}
	return false
}

func queryIdent(b byte) bool {
	return b == '_' || b == '$' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

func skipWhitespace(q string, i int) int {
	for i < len(q) {
		switch {
		case querySpace(q[i]):
			i++
		case q[i] == '-' && i+1 < len(q) && q[i+1] == '-':
			i = skipLineComment(q, i+2)
		case q[i] == '/' && i+1 < len(q) && q[i+1] == '*':
			i = skipBlockComment(q, i+2)
		default:
			return i
		}
	}
	return i
}

func skipLineComment(q string, i int) int {
	for i < len(q) && q[i] != '\n' {
		i++
	}
	return i
}

func skipBlockComment(q string, i int) int {
	for i+1 < len(q) && (q[i] != '*' || q[i+1] != '/') {
		i++
	}
	return i + 2
}

func skipQueryString(q string, i int) int {
	quote := q[i]
	for i++; i < len(q); i++ {
		if q[i] == quote {
			if i+1 < len(q) && q[i+1] == quote {
				i++
				continue
			}
			return i + 1
		}
	}
	return i
}

func formatRows(rows *sql.Rows, cellLen int) (string, error) {
	cols, err := rows.ColumnTypes()
	if err != nil {
		return "", fmt.Errorf("read columns: %w", err)
	}
	if len(cols) == 0 {
		return "no rows", nil
	}
	header := make([]string, len(cols))
	for i, col := range cols {
		header[i] = col.Name()
	}
	dest := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range dest {
		ptrs[i] = &dest[i]
	}
	var data [][]string
	var more, truncated int
	for rows.Next() {
		if len(data) >= maxRows {
			more++
			continue
		}
		if err := rows.Scan(ptrs...); err != nil {
			return "", fmt.Errorf("scan row: %w", err)
		}
		row := make([]string, len(cols))
		for i, value := range dest {
			row[i] = renderCell(value)
			if len(row[i]) > cellLen {
				truncated++
			}
		}
		data = append(data, row)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("read rows: %w", err)
	}
	if len(data) == 0 {
		return "no rows", nil
	}
	widths := make([]int, len(cols))
	for i, value := range header {
		widths[i] = len(display(cellLen, value))
	}
	for _, row := range data {
		for i, value := range row {
			if width := len(display(cellLen, value)); width > widths[i] {
				widths[i] = width
			}
		}
	}
	var b strings.Builder
	writeRow := func(row []string) {
		for i, value := range row {
			if i > 0 {
				b.WriteString("  ")
			}
			fmt.Fprintf(&b, "%-*s", widths[i], display(cellLen, value))
		}
		b.WriteByte('\n')
	}
	writeRow(header)
	for i, width := range widths {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(strings.Repeat("-", width))
	}
	b.WriteByte('\n')
	for _, row := range data {
		writeRow(row)
	}
	if truncated > 0 {
		fmt.Fprintf(&b, "... (%d cells truncated; use length()/substr() to read full values)\n", truncated)
	}
	if more > 0 {
		fmt.Fprintf(&b, "... (%d more rows)", more)
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

func renderCell(value any) string {
	switch value := value.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(value)
	case time.Time:
		return value.Format(time.DateTime)
	default:
		return fmt.Sprint(value)
	}
}

func display(cellLen int, value string) string {
	if len(value) <= cellLen {
		return value
	}
	return value[:cellLen] + "..."
}
