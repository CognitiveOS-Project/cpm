package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSecrets(t *testing.T) {
	s := NewSecrets()
	if s.Secrets == nil {
		t.Fatal("expected non-nil Secrets map")
	}
	if len(s.Secrets) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(s.Secrets))
	}
}

func TestSaveAndLoadGlobal(t *testing.T) {
	dir := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", orig)

	s := NewSecrets()
	s.Secrets["API_KEY"] = "test-value-123"

	if err := SaveSecrets("global", "", s); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadSecrets("global", "")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Secrets["API_KEY"] != "test-value-123" {
		t.Fatalf("expected test-value-123, got %q", loaded.Secrets["API_KEY"])
	}

	// Verify file permissions
	path := filepath.Join(dir, ".cpm", SecretsFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}

func TestSaveAndLoadPatch(t *testing.T) {
	patchesDir := t.TempDir()
	os.Setenv("CPM_PATCHES_DIR", filepath.Join(patchesDir, "patches"))
	defer os.Unsetenv("CPM_PATCHES_DIR")

	s := NewSecrets()
	s.Secrets["PATCH_SECRET"] = "patch-value"

	if err := SaveSecrets("patch", "my-skill", s); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadSecrets("patch", "my-skill")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Secrets["PATCH_SECRET"] != "patch-value" {
		t.Fatalf("expected patch-value, got %q", loaded.Secrets["PATCH_SECRET"])
	}
}

func TestLoadReturnsEmptyWhenNotExist(t *testing.T) {
	dir := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", orig)

	s, err := LoadSecrets("global", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(s.Secrets) != 0 {
		t.Fatalf("expected empty secrets, got %d", len(s.Secrets))
	}
}

func TestMergeSecrets(t *testing.T) {
	global := &Secrets{Secrets: map[string]string{
		"GLOBAL_KEY":  "global-val",
		"SHARED_KEY":  "global-shared",
	}}
	patch := &Secrets{Secrets: map[string]string{
		"PATCH_KEY":   "patch-val",
		"SHARED_KEY":  "patch-shared",
	}}

	merged := MergeSecrets(global, patch)

	if merged.Secrets["GLOBAL_KEY"] != "global-val" {
		t.Errorf("GLOBAL_KEY = %q, want global-val", merged.Secrets["GLOBAL_KEY"])
	}
	if merged.Secrets["PATCH_KEY"] != "patch-val" {
		t.Errorf("PATCH_KEY = %q, want patch-val", merged.Secrets["PATCH_KEY"])
	}
	if merged.Secrets["SHARED_KEY"] != "patch-shared" {
		t.Errorf("SHARED_KEY = %q, want patch-shared (patch overrides global)", merged.Secrets["SHARED_KEY"])
	}
}

func TestMergeSecretsNilPatch(t *testing.T) {
	global := &Secrets{Secrets: map[string]string{"A": "1"}}
	merged := MergeSecrets(global, nil)
	if merged.Secrets["A"] != "1" {
		t.Errorf("expected A=1, got %q", merged.Secrets["A"])
	}
}

func TestMergeSecretsNilGlobal(t *testing.T) {
	patch := &Secrets{Secrets: map[string]string{"B": "2"}}
	merged := MergeSecrets(nil, patch)
	if merged.Secrets["B"] != "2" {
		t.Errorf("expected B=2, got %q", merged.Secrets["B"])
	}
}

func TestRemoveSecret(t *testing.T) {
	patchesDir := t.TempDir()
	os.Setenv("CPM_PATCHES_DIR", filepath.Join(patchesDir, "patches"))
	defer os.Unsetenv("CPM_PATCHES_DIR")

	s := NewSecrets()
	s.Secrets["KEY_A"] = "val-a"
	s.Secrets["KEY_B"] = "val-b"

	if err := SaveSecrets("patch", "test-pkg", s); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := RemoveSecret("patch", "test-pkg", "KEY_A"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	loaded, err := LoadSecrets("patch", "test-pkg")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if _, ok := loaded.Secrets["KEY_A"]; ok {
		t.Error("KEY_A should have been removed")
	}
	if loaded.Secrets["KEY_B"] != "val-b" {
		t.Errorf("KEY_B = %q, want val-b", loaded.Secrets["KEY_B"])
	}
}

func TestRemoveSecretNotFound(t *testing.T) {
	patchesDir := t.TempDir()
	os.Setenv("CPM_PATCHES_DIR", filepath.Join(patchesDir, "patches"))
	defer os.Unsetenv("CPM_PATCHES_DIR")

	s := NewSecrets()
	s.Secrets["ONLY_KEY"] = "val"
	if err := SaveSecrets("patch", "test-pkg", s); err != nil {
		t.Fatalf("save: %v", err)
	}

	err := RemoveSecret("patch", "test-pkg", "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for nonexistent secret")
	}
}

func TestPatchScopeRequiresName(t *testing.T) {
	_, err := LoadSecrets("patch", "")
	if err == nil {
		t.Fatal("expected error for empty patch name")
	}
}

func TestUnknownScope(t *testing.T) {
	_, err := LoadSecrets("invalid", "")
	if err == nil {
		t.Fatal("expected error for invalid scope")
	}
}

func TestDeleteSecrets(t *testing.T) {
	patchesDir := t.TempDir()
	os.Setenv("CPM_PATCHES_DIR", filepath.Join(patchesDir, "patches"))
	defer os.Unsetenv("CPM_PATCHES_DIR")

	s := NewSecrets()
	s.Secrets["KEY"] = "val"
	if err := SaveSecrets("patch", "del-pkg", s); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := DeleteSecrets("patch", "del-pkg"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	loaded, err := LoadSecrets("patch", "del-pkg")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Secrets) != 0 {
		t.Errorf("expected empty secrets after delete, got %d", len(loaded.Secrets))
	}
}
