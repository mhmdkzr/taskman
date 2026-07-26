package pg

import (
	"strings"
	"testing"
)

func TestOpen_ReturnsPingError(t *testing.T) {
	_, err := Open(t.Context(), Config{
		Host:     "127.0.0.1",
		Port:     1,
		User:     "user",
		Password: "pass",
		Database: "core",
		SSLMode:  "disable",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to ping database") {
		t.Fatalf("unexpected error: %v", err)
	}
}
