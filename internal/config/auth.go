package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type AuthConfig struct {
	KeyPath      string `json:"key_path"`
	Fingerprint  string `json:"fingerprint"`
	Registered   bool   `json:"registered"`
	RegisteredAt string `json:"registered_at,omitempty"`
	LastVerified string `json:"last_verified"`
}

func AuthPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cpm", "auth.json")
}

func LoadAuth() (*AuthConfig, error) {
	path := AuthPath()
	if path == "" {
		return nil, fmt.Errorf("cannot determine home directory")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in (run: cpm auth login)")
		}
		return nil, fmt.Errorf("read auth config: %w", err)
	}

	var cfg AuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse auth config: %w", err)
	}

	return &cfg, nil
}

func SaveAuth(cfg *AuthConfig) error {
	path := AuthPath()
	if path == "" {
		return fmt.Errorf("cannot determine home directory")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	cfg.LastVerified = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal auth config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write auth config: %w", err)
	}

	return nil
}

func RemoveAuth() error {
	path := AuthPath()
	if path == "" {
		return nil
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove auth config: %w", err)
	}

	return nil
}
