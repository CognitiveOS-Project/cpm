package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CognitiveOS-Project/cpm/internal/config"
	"github.com/CognitiveOS-Project/cpm/internal/registry"
)

func TestLoginStoresKeyPath(t *testing.T) {
	home := t.TempDir()
	authPath := filepath.Join(home, ".cpm", "auth.json")

	keyPath := createTempSSHKey(t)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", origHome)

	// Mock registry
	registryClient = &mockRegistry{
		authStatusFunc: func(fingerprint string) (*registry.AuthStatusResponse, error) {
			return &registry.AuthStatusResponse{
				Fingerprint: fingerprint,
				Registered:  true,
			}, nil
		},
	}
	defer func() { registryClient = nil }()

	rootCmd.SetArgs([]string{"auth", "login", "--key", keyPath})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if _, err := os.Stat(authPath); err != nil {
		t.Fatalf("auth.json not created: %v", err)
	}

	cfg, err := config.LoadAuth()
	if err != nil {
		t.Fatalf("LoadAuth failed: %v", err)
	}
	if cfg.KeyPath != keyPath {
		t.Errorf("expected key_path %s, got %s", keyPath, cfg.KeyPath)
	}
	if !cfg.Registered {
		t.Errorf("expected registered=true, got false")
	}
}

func TestLogoutRemovesAuth(t *testing.T) {
	home := t.TempDir()
	authPath := filepath.Join(home, ".cpm", "auth.json")

	os.MkdirAll(filepath.Dir(authPath), 0700)
	os.WriteFile(authPath, []byte(`{"key_path":"/tmp/key","fingerprint":"SHA256:test"}`), 0600)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", origHome)

	rootCmd.SetArgs([]string{"auth", "logout"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	if _, err := os.Stat(authPath); !os.IsNotExist(err) {
		t.Errorf("auth.json should be removed after logout")
	}
}

func TestLoginNotRegistered(t *testing.T) {
	home := t.TempDir()
	keyPath := createTempSSHKey(t)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", origHome)

	registryClient = &mockRegistry{
		authStatusFunc: func(fingerprint string) (*registry.AuthStatusResponse, error) {
			return &registry.AuthStatusResponse{
				Fingerprint: fingerprint,
				Registered:  false,
			}, nil
		},
	}
	defer func() { registryClient = nil }()

	rootCmd.SetArgs([]string{"auth", "login", "--key", keyPath})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	cfg, err := config.LoadAuth()
	if err != nil {
		t.Fatalf("LoadAuth failed: %v", err)
	}
	if cfg.Registered {
		t.Errorf("expected registered=false, got true")
	}
}
