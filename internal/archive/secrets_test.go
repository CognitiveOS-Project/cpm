package archive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/CognitiveOS-Project/cpm/internal/config"
)

func setupPatchSecrets(t *testing.T, patchName string, secrets map[string]string) string {
	t.Helper()
	patchesDir := t.TempDir()
	os.Setenv("CPM_PATCHES_DIR", filepath.Join(patchesDir, "patches"))
	t.Cleanup(func() { os.Unsetenv("CPM_PATCHES_DIR") })

	s := config.NewSecrets()
	s.Secrets = secrets
	if err := config.SaveSecrets("patch", patchName, s); err != nil {
		t.Fatalf("save patch secrets: %v", err)
	}
	return patchesDir
}

func setupGlobalSecrets(t *testing.T, secrets map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Unsetenv("HOME") })

	s := config.NewSecrets()
	s.Secrets = secrets
	if err := config.SaveSecrets("global", "", s); err != nil {
		t.Fatalf("save global secrets: %v", err)
	}
	return dir
}

func TestResolveSecretsSingle(t *testing.T) {
	setupGlobalSecrets(t, map[string]string{"API_KEY": "sk-abc123"})

	input := []byte(`{"api_key": "${API_KEY}"}`)
	resolved, err := ResolveSecrets(input, "global", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	var result map[string]string
	json.Unmarshal(resolved, &result)

	if result["api_key"] != "sk-abc123" {
		t.Errorf("api_key = %q, want sk-abc123", result["api_key"])
	}
}

func TestResolveSecretsMultiple(t *testing.T) {
	setupGlobalSecrets(t, map[string]string{
		"API_KEY":    "key-123",
		"API_SECRET": "secret-456",
	})

	input := []byte(`{"key": "${API_KEY}", "secret": "${API_SECRET}"}`)
	resolved, err := ResolveSecrets(input, "global", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	var result map[string]string
	json.Unmarshal(resolved, &result)

	if result["key"] != "key-123" {
		t.Errorf("key = %q, want key-123", result["key"])
	}
	if result["secret"] != "secret-456" {
		t.Errorf("secret = %q, want secret-456", result["secret"])
	}
}

func TestResolveSecretsNoPlaceholders(t *testing.T) {
	input := []byte(`{"key": "literal-value"}`)
	resolved, err := ResolveSecrets(input, "global", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if string(resolved) != string(input) {
		t.Errorf("expected unchanged data")
	}
}

func TestResolveSecretsUndefined(t *testing.T) {
	setupGlobalSecrets(t, map[string]string{})

	input := []byte(`{"key": "${UNDEFINED_SECRET}"}`)
	_, err := ResolveSecrets(input, "global", "")
	if err == nil {
		t.Fatal("expected error for undefined secret")
	}
}

func TestResolveSecretsPatchOverridesGlobal(t *testing.T) {
	setupGlobalSecrets(t, map[string]string{"API_KEY": "global-val"})
	setupPatchSecrets(t, "my-pkg", map[string]string{"API_KEY": "patch-val"})

	input := []byte(`{"key": "${API_KEY}"}`)
	resolved, err := ResolveSecrets(input, "patch", "my-pkg")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	var result map[string]string
	json.Unmarshal(resolved, &result)

	if result["key"] != "patch-val" {
		t.Errorf("api_key = %q, want patch-val (patch overrides global)", result["key"])
	}
}

func TestResolveManifest(t *testing.T) {
	setupGlobalSecrets(t, map[string]string{"OPENAI_KEY": "sk-real"})

	m := &Manifest{
		Name:    "test-pkg",
		Version: "1.0.0",
		Brain: &BrainConfig{
			WideModel: &ModelConfig{
				Weights: &WeightsConfig{
					Cloud: &CloudWeights{
						APIBase: "https://api.openai.com/v1",
						ModelID: "gpt-4o-mini",
						APIKey:  "${OPENAI_KEY}",
					},
				},
			},
		},
	}

	if err := ResolveManifest(m, "global", ""); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if m.Brain.WideModel.Weights.Cloud.APIKey != "sk-real" {
		t.Errorf("api_key = %q, want sk-real", m.Brain.WideModel.Weights.Cloud.APIKey)
	}
}

func TestResolveSecretsDuplicatePlaceholder(t *testing.T) {
	setupGlobalSecrets(t, map[string]string{"KEY": "val"})

	input := []byte(`{"a": "${KEY}", "b": "${KEY}"}`)
	resolved, err := ResolveSecrets(input, "global", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	var result map[string]string
	json.Unmarshal(resolved, &result)

	if result["a"] != "val" || result["b"] != "val" {
		t.Errorf("expected both values to be 'val', got a=%q b=%q", result["a"], result["b"])
	}
}
