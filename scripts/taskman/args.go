package main

import (
	"fmt"
	"os"
	"strings"
)

// ---------------------------------------------------------------------
// shared flag helpers
// ---------------------------------------------------------------------

// repeatedFlag collects one value per occurrence of a flag, e.g.
// --question "a" --question "b" -> ["a", "b"]. Unlike splitCSV's
// comma-separated flags, each value can itself contain commas.
type repeatedFlag []string

func (r *repeatedFlag) String() string { return strings.Join(*r, "; ") }
func (r *repeatedFlag) Set(v string) error {
	*r = append(*r, v)
	return nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// textOrFile resolves a "--x"/"--x-file" flag pair: --x-file wins if both
// are set to a non-empty value.
func textOrFile(text, file string) (string, error) {
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return text, nil
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "taskman: "+format+"\n", args...)
	os.Exit(2)
}
