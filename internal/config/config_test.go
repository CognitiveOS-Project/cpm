package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWhenNoFile(t *testing.T) {
	r, err := Load("/nonexistent/path.toml")
	if err != nil {
		t.Fatalf("Load should not error when file missing: %v", err)
	}
	if r.Official.Primary != "https://registry-us-all-distros-official.cognitive-os.org/v1" {
		t.Fatalf("expected default primary, got %s", r.Official.Primary)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registries.toml")
	os.WriteFile(path, []byte(`
[official]
primary = "https://registry-us-all-distros-official.cognitive-os.org/v1"

[official.mirrors]
eu = "https://registry-eu-all-distros-official.cognitive-os.org/v1"
jp = "https://registry-jp-all-distros-official.cognitive-os.org/v1"

[alternative]
community = "https://community-registry.cognitive-os.org/v1"
my-private = "https://my-registry.example.com/v1"
`), 0644)

	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if r.Official.Primary != "https://registry-us-all-distros-official.cognitive-os.org/v1" {
		t.Fatalf("expected primary, got %s", r.Official.Primary)
	}

	if len(r.Official.Mirrors) != 2 {
		t.Fatalf("expected 2 mirrors, got %d", len(r.Official.Mirrors))
	}
	if r.Official.Mirrors["eu"] != "https://registry-eu-all-distros-official.cognitive-os.org/v1" {
		t.Fatalf("expected eu mirror, got %s", r.Official.Mirrors["eu"])
	}

	if r.Alternatives["community"] != "https://community-registry.cognitive-os.org/v1" {
		t.Fatalf("expected community registry, got %s", r.Alternatives["community"])
	}
}

func TestResolve(t *testing.T) {
	r, _ := Load("/nonexistent/path.toml")

	url, err := r.Resolve("")
	if err != nil {
		t.Fatalf("Resolve empty should not error: %v", err)
	}
	if url != "https://registry-us-all-distros-official.cognitive-os.org/v1" {
		t.Fatalf("expected default primary, got %s", url)
	}

	url, err = r.Resolve("official")
	if err != nil {
		t.Fatalf("Resolve official should not error: %v", err)
	}
	if url != "https://registry-us-all-distros-official.cognitive-os.org/v1" {
		t.Fatalf("expected official primary, got %s", url)
	}
}

func TestResolveCustom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registries.toml")
	os.WriteFile(path, []byte(`
[official]
primary = "https://primary.example.com/v1"

[official.mirrors]
eu = "https://eu.example.com/v1"

[alternative]
mine = "https://mine.example.com/v1"
`), 0644)

	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	url, err := r.Resolve("official.eu")
	if err != nil {
		t.Fatalf("Resolve official.eu: %v", err)
	}
	if url != "https://eu.example.com/v1" {
		t.Fatalf("expected eu mirror, got %s", url)
	}

	url, err = r.Resolve("alternative.mine")
	if err != nil {
		t.Fatalf("Resolve alternative.mine: %v", err)
	}
	if url != "https://mine.example.com/v1" {
		t.Fatalf("expected mine, got %s", url)
	}

	_, err = r.Resolve("official.bogus")
	if err == nil {
		t.Fatal("expected error for unknown mirror")
	}

	_, err = r.Resolve("alternative")
	if err == nil {
		t.Fatal("expected error for alternative without name")
	}

	_, err = r.Resolve("bogus")
	if err == nil {
		t.Fatal("expected error for unknown section")
	}
}
