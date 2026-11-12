package patch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPatchList(t *testing.T) {
	// Create a temporary patches directory with a fake patch
	tmpDir := t.TempDir()
	patchDir := filepath.Join(tmpDir, "test-patch")
	if err := os.MkdirAll(patchDir, 0755); err != nil {
		t.Fatalf("failed to create patch dir: %v", err)
	}
	// Create a dummy cognitive.json
	manifest := `{"name":"test-patch","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(patchDir, "cognitive.json"), []byte(manifest), 0644); err != nil {
		t.Fatalf("failed to create manifest: %v", err)
	}

	origPatchesDir := PatchesDir
	PatchesDir = tmpDir
	defer func() { PatchesDir = origPatchesDir }()

	patches, err := List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Manifest.Name != "test-patch" {
		t.Fatalf("expected patch name test-patch, got %s", patches[0].Manifest.Name)
	}
}

func TestPatchReadManifest(t *testing.T) {
	tmpDir := t.TempDir()
	patchDir := filepath.Join(tmpDir, "test-patch")
	if err := os.MkdirAll(patchDir, 0755); err != nil {
		t.Fatalf("failed to create patch dir: %v", err)
	}
	manifest := `{"name":"test-patch","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(patchDir, "cognitive.json"), []byte(manifest), 0644); err != nil {
		t.Fatalf("failed to create manifest: %v", err)
	}

	origPatchesDir := PatchesDir
	PatchesDir = tmpDir
	defer func() { PatchesDir = origPatchesDir }()

	m, err := ReadManifest("test-patch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "test-patch" {
		t.Fatalf("expected name test-patch, got %s", m.Name)
	}
}

func TestPatchIsInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	patchDir := filepath.Join(tmpDir, "test-patch")
	if err := os.MkdirAll(patchDir, 0755); err != nil {
		t.Fatalf("failed to create patch dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(patchDir, "cognitive.json"), []byte(`{"name":"test-patch"}`), 0644); err != nil {
		t.Fatalf("failed to create manifest: %v", err)
	}

	origPatchesDir := PatchesDir
	PatchesDir = tmpDir
	defer func() { PatchesDir = origPatchesDir }()

	if !IsInstalled("test-patch") {
		t.Fatal("expected test-patch to be installed")
	}
	if IsInstalled("nonexistent-patch") {
		t.Fatal("expected nonexistent-patch to not be installed")
	}
}

func TestPatchDir(t *testing.T) {
	tmpDir := t.TempDir()
	patchName := "test-patch"
	expectedPath := filepath.Join(tmpDir, patchName)

	origPatchesDir := PatchesDir
	PatchesDir = tmpDir
	defer func() { PatchesDir = origPatchesDir }()

	got := Dir(patchName)
	if got != expectedPath {
		t.Fatalf("expected Dir(%q) = %q, got %q", patchName, expectedPath, got)
	}
}

func TestPatchRemove(t *testing.T) {
	tmpDir := t.TempDir()
	patchDir := filepath.Join(tmpDir, "test-patch")
	if err := os.MkdirAll(patchDir, 0755); err != nil {
		t.Fatalf("failed to create patch dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(patchDir, "cognitive.json"), []byte(`{"name":"test-patch"}`), 0644); err != nil {
		t.Fatalf("failed to create manifest: %v", err)
	}

	origPatchesDir := PatchesDir
	PatchesDir = tmpDir
	defer func() { PatchesDir = origPatchesDir }()

	if err := Remove("test-patch"); err != nil {
		t.Fatalf("unexpected error removing patch: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "test-patch")); !os.IsNotExist(err) {
		t.Fatal("expected patch dir to be removed after Remove")
	}
}