package testenv

import (
	"bufio"
	"io"
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
		slog.Error("failed to open .env file", "path", path, "error", err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close .env file", "error", err)
		}
	}()

	dotEnv = parseDotEnv(f)
}

// parseDotEnv parses .env content into a key/value map. Blank lines and lines
// whose first non-whitespace character is '#' are skipped; an optional
// "export " prefix is stripped; keys and values are trimmed of surrounding
// whitespace; surrounding quotes ('"' or '\”) are removed from values;
// malformed lines without '=' and empty keys are ignored; values containing
// '=' preserve everything after the first '='. Inline '#' text is kept as
// part of the value.
func parseDotEnv(r io.Reader) map[string]string {
	out := map[string]string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if after, ok := strings.CutPrefix(line, "export "); ok {
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
			out[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		slog.Error("failed to scan .env file", "error", err)
	}
	return out
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
