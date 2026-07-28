package archive

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/CognitiveOS-Project/cpm/internal/config"
)

var secretPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// ResolveSecrets replaces ${VAR} placeholders in data with secret values.
// scope and patchName determine which secrets store to use.
// Returns an error if any placeholder references an undefined secret.
func ResolveSecrets(data []byte, scope, patchName string) ([]byte, error) {
	matches := secretPattern.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return data, nil
	}

	global, err := config.LoadSecrets("global", "")
	if err != nil {
		return nil, fmt.Errorf("load global secrets: %w", err)
	}

	var patchSecrets *config.Secrets
	if scope == "patch" && patchName != "" {
		patchSecrets, err = config.LoadSecrets("patch", patchName)
		if err != nil {
			return nil, fmt.Errorf("load patch secrets: %w", err)
		}
	}

	merged := config.MergeSecrets(global, patchSecrets)

	// Collect all referenced variable names and check they exist
	seen := make(map[string]bool)
	for _, match := range matches {
		name := string(match[1])
		if seen[name] {
			continue
		}
		seen[name] = true
		if _, ok := merged.Secrets[name]; !ok {
			return nil, fmt.Errorf("secret %q not found — add it with: cpm secret add %s <value>", name, name)
		}
	}

	// Replace all placeholders
	result := secretPattern.ReplaceAllStringFunc(string(data), func(match string) string {
		submatch := secretPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		return merged.Secrets[submatch[1]]
	})

	return []byte(result), nil
}

// ResolveManifest resolves ${VAR} placeholders in a manifest in-place.
// It marshals the manifest to JSON, resolves secrets, and unmarshals back.
func ResolveManifest(m *Manifest, scope, patchName string) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	resolved, err := ResolveSecrets(data, scope, patchName)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(resolved, m); err != nil {
		return fmt.Errorf("unmarshal resolved manifest: %w", err)
	}

	return nil
}
