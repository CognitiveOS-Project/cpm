package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWhenNoFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path.toml")
	if err != nil {
		t.Fatalf("Load should not error when file missing: %v", err)
	}
	if cfg.DefaultRegistry != "https://registry.cognitive-os.org/v1" {
		t.Fatalf("expected default registry, got %s", cfg.DefaultRegistry)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registries.toml")
	os.WriteFile(path, []byte(`
[default]
url = "https://my-registry.example.com/v1"
`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.DefaultRegistry != "https://my-registry.example.com/v1" {
		t.Fatalf("expected custom registry, got %s", cfg.DefaultRegistry)
	}
}
