package read

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDirectoryExcludesBinaryDocuments(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.go", "image.png", "manual.pdf", "subdir"} {
		path := filepath.Join(dir, name)
		if filepath.Ext(name) == "" {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := execute(t.Context(), Input{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(dir, "a.go"), filepath.Join(dir, "subdir")}
	if len(got.Entries) != len(want) {
		t.Fatalf("entries = %v, want %v", got.Entries, want)
	}
	for i := range want {
		if got.Entries[i] != want[i] {
			t.Fatalf("entries = %v, want %v", got.Entries, want)
		}
	}
}
