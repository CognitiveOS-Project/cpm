package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const SecretsFileName = "secrets.json"

// Secrets holds key-value secret pairs.
type Secrets struct {
	Secrets map[string]string `json:"secrets"`
}

// NewSecrets creates an empty Secrets instance.
func NewSecrets() *Secrets {
	return &Secrets{Secrets: make(map[string]string)}
}

// patchSecretsDir returns /cognitiveos/lib/cpm/secrets/<patchName>/
func patchSecretsDir(patchName string) string {
	if d := os.Getenv("CPM_PATCHES_DIR"); d != "" {
		return filepath.Join(filepath.Dir(d), "lib", "cpm", "secrets", patchName)
	}
	return filepath.Join("/cognitiveos/lib/cpm/secrets", patchName)
}

// globalSecretsDir returns ~/.cpm/
func globalSecretsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cpm")
}

// LoadSecrets loads secrets from the specified scope.
// scope must be "patch" or "global".
// For "patch" scope, patchName is required.
// Returns an empty Secrets (no error) if the file doesn't exist.
func LoadSecrets(scope, patchName string) (*Secrets, error) {
	var dir string

	switch scope {
	case "patch":
		if patchName == "" {
			return nil, fmt.Errorf("patch name required for patch scope")
		}
		dir = patchSecretsDir(patchName)
	case "global":
		dir = globalSecretsDir()
		if dir == "" {
			return nil, fmt.Errorf("cannot determine home directory")
		}
	default:
		return nil, fmt.Errorf("unknown scope %q (use 'patch' or 'global')", scope)
	}

	path := filepath.Join(dir, SecretsFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewSecrets(), nil
		}
		return nil, fmt.Errorf("read secrets: %w", err)
	}

	var s Secrets
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse secrets: %w", err)
	}

	if s.Secrets == nil {
		s.Secrets = make(map[string]string)
	}

	return &s, nil
}

// SaveSecrets writes secrets to the specified scope.
func SaveSecrets(scope, patchName string, s *Secrets) error {
	var dir string

	switch scope {
	case "patch":
		if patchName == "" {
			return fmt.Errorf("patch name required for patch scope")
		}
		dir = patchSecretsDir(patchName)
	case "global":
		dir = globalSecretsDir()
		if dir == "" {
			return fmt.Errorf("cannot determine home directory")
		}
	default:
		return fmt.Errorf("unknown scope %q", scope)
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create secrets directory: %w", err)
	}

	path := filepath.Join(dir, SecretsFileName)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal secrets: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write secrets: %w", err)
	}

	return nil
}

// DeleteSecrets removes the secrets file for the specified scope.
func DeleteSecrets(scope, patchName string) error {
	var dir string

	switch scope {
	case "patch":
		if patchName == "" {
			return fmt.Errorf("patch name required for patch scope")
		}
		dir = patchSecretsDir(patchName)
	case "global":
		dir = globalSecretsDir()
		if dir == "" {
			return nil
		}
	default:
		return fmt.Errorf("unknown scope %q", scope)
	}

	path := filepath.Join(dir, SecretsFileName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove secrets: %w", err)
	}

	return nil
}

// MergeSecrets returns a new Secrets with global values overridden by patch values.
func MergeSecrets(global, patch *Secrets) *Secrets {
	merged := NewSecrets()

	if global != nil {
		for k, v := range global.Secrets {
			merged.Secrets[k] = v
		}
	}

	if patch != nil {
		for k, v := range patch.Secrets {
			merged.Secrets[k] = v
		}
	}

	return merged
}

// RemoveSecret removes a single key from the secrets file.
func RemoveSecret(scope, patchName, name string) error {
	s, err := LoadSecrets(scope, patchName)
	if err != nil {
		return err
	}

	if _, ok := s.Secrets[name]; !ok {
		return fmt.Errorf("secret %q not found in %s scope", name, scope)
	}

	delete(s.Secrets, name)
	return SaveSecrets(scope, patchName, s)
}
