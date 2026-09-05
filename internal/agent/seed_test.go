package agent

import (
	"encoding/json"
	"testing"
)

func TestToolSeedsMatchRegistry(t *testing.T) {
	registry := Tools()
	seeds := toolSeeds()
	if len(seeds) != len(registry) {
		t.Fatalf("seed count = %d, want %d", len(seeds), len(registry))
	}

	for _, seed := range seeds {
		if seed.Name == "" {
			t.Fatal("tool seed name is empty")
		}
		if seed.Description == "" {
			t.Fatalf("tool %q description is empty", seed.Name)
		}
		if _, ok := registry[seed.Name]; !ok {
			t.Fatalf("tool %q is not registered", seed.Name)
		}
		if !json.Valid(seed.InputSchema) {
			t.Fatalf("tool %q input schema is invalid JSON", seed.Name)
		}
		if !json.Valid(seed.OutputSchema) {
			t.Fatalf("tool %q output schema is invalid JSON", seed.Name)
		}
	}
}

func TestProviderNamesAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(providerNames))
	for _, name := range providerNames {
		if name == "" {
			t.Fatal("provider name is empty")
		}
		if _, ok := seen[name]; ok {
			t.Fatalf("provider %q is seeded more than once", name)
		}
		seen[name] = struct{}{}
	}
	if _, ok := seen[currentProviderName]; !ok {
		t.Fatalf("current provider %q is not seeded", currentProviderName)
	}
}
