package auditlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// insertAuditLog inserts an audit event into the database.
func insertAuditLog(ctx context.Context, db *sql.DB, event EventAPIAuditLogged) error {
	requestHeaders, err := json.Marshal(event.Request.Headers)
	if err != nil {
		return fmt.Errorf("marshal request headers: %w", err)
	}
	responseHeaders, err := json.Marshal(event.Response.Headers)
	if err != nil {
		return fmt.Errorf("marshal response headers: %w", err)
	}

	requestBody := toJSONBCompatibleString(event.Request.Body)
	responseBody := toJSONBCompatibleString(event.Response.Body)

	_, err = db.ExecContext(ctx, `
		INSERT INTO audit_logs (
			request_id,
			timestamp,
			duration_ns,
			request_method,
			request_path,
			request_query,
			request_remote_addr,
			request_user_agent,
			request_proto,
			request_headers,
			request_body,
			response_status_code,
			response_headers,
			response_body,
			response_size
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, $12, $13::jsonb, $14::jsonb, $15)
		ON CONFLICT (request_id) DO NOTHING`,
		event.RequestID,
		event.Timestamp,
		event.Duration.Nanoseconds(),
		event.Request.Method,
		event.Request.Path,
		event.Request.Query,
		event.Request.RemoteAddr,
		event.Request.UserAgent,
		event.Request.Proto,
		requestHeaders,
		requestBody,
		event.Response.StatusCode,
		responseHeaders,
		responseBody,
		event.Response.Size,
	)
	if err != nil {
		return fmt.Errorf("insert audit log %s: %w", event.RequestID, err)
	}
	return nil
}

// toJSONBCompatibleString converts a byte slice to a JSONB-compatible string pointer.
func toJSONBCompatibleString(body []byte) *string {
	if len(body) == 0 {
		return nil
	}
	var value any
	if json.Unmarshal(body, &value) == nil {
		removeNUL(value)
		if encoded, err := json.Marshal(value); err == nil {
			s := string(encoded)
			return &s
		}
	}
	b, err := json.Marshal(strings.ReplaceAll(string(body), "\x00", ""))
	if err != nil {
		s := strings.ReplaceAll(string(body), "\x00", "")
		return &s
	}
	s := string(b)
	return &s
}

// removeNUL removes code points PostgreSQL JSONB does not accept.
func removeNUL(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			switch child := nested.(type) {
			case string:
				typed[key] = strings.ReplaceAll(child, "\x00", "")
			default:
				removeNUL(child)
			}
		}
	case []any:
		for i, nested := range typed {
			switch child := nested.(type) {
			case string:
				typed[i] = strings.ReplaceAll(child, "\x00", "")
			default:
				removeNUL(child)
			}
		}
	}
}
