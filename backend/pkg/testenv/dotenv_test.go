package testenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDotEnvValueAndEnvOrDefault(t *testing.T) {
	tmp := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(tmp, ".env"),
		[]byte("export APP_SAMPLE='from-dotenv'\nAPP_OTHER=other\n"),
		0o600,
	); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	t.Chdir(tmp)

	if got := DotEnvValue("APP_SAMPLE"); got != "from-dotenv" {
		t.Fatalf("dotenv value mismatch: %q", got)
	}
	t.Setenv("APP_OTHER", "from-env")
	if got := EnvOrDefault("APP_OTHER", "fallback"); got != "from-env" {
		t.Fatalf("env override mismatch: %q", got)
	}
	if got := EnvOrDefault("APP_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("fallback mismatch: %q", got)
	}
}
