package testenv

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	loadOnce sync.Once
	dotEnv   map[string]string
)

// EnvOrDefault returns the environment variable value or falls back to a .env file or default.
func EnvOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	if v := DotEnvValue(key); v != "" {
		return v
	}
	return fallback
}

// DotEnvValue returns the value for a key from the .env file.
func DotEnvValue(key string) string {
	loadOnce.Do(loadDotEnv)
	if v, ok := dotEnv[key]; ok {
		return v
	}
	return ""
}

// loadDotEnv loads environment variables from the nearest .env file.
func loadDotEnv() {
	dotEnv = map[string]string{}
	path, ok := findDotEnvPath()
	if !ok {
		return
	}

	f, err := os.Open(path) // #nosec G304 -- path comes from walking parent directories to find the repo .env file.
	if err != nil {
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close .env file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if after, ok0 := strings.CutPrefix(line, "export "); ok0 {
			line = strings.TrimSpace(after)
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}
		if key != "" {
			dotEnv[key] = value
		}
	}
}

// findDotEnvPath searches up the directory tree for a .env file.
func findDotEnvPath() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, ".env")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
