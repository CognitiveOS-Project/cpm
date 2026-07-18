package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/auth"
	"github.com/CognitiveOS-Project/cpm/internal/registry"
	"github.com/spf13/cobra"
)

var (
	publishDownloadURL string
	publishTags       []string
	publishScope      string
	publishVisibility string
	publishKeyPath    string
)

var registryClient registry.Registry

func getRegistryClient() registry.Registry {
	if registryClient != nil {
		return registryClient
	}
	return registry.New(resolveRegistry())
}

func defaultSSHKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "id_ed25519")
}

var publishCmd = &cobra.Command{
	Use:   "publish <path>",
	Short: "Publish a .cgp patch to the registry",
	Long: `Publish a .cgp archive to the configured registry notary.

Authenticates using SSH key signing. The manifest SHA-256 is signed
with the publisher's private key. The registry verifies the signature
against the registered public key.

If no SSH key is found, falls back to CPM_REGISTRY_TOKEN (deprecated).

Examples:
  cpm publish ./my-patch-1.0.0.cgp --download-url https://github.com/.../my-patch-1.0.0.cgp
  cpm publish ./my-patch-1.0.0.cgp --key ~/.ssh/my_key --download-url https://...
  cpm publish ./my-patch-1.0.0.cgp --tag vision --download-url https://...`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("ERROR:P001: open %s: %w", path, err)
		}
		defer f.Close()

		m, err := archive.ReadManifest(f)
		if err != nil {
			return fmt.Errorf("ERROR:P002: read manifest: %w", err)
		}

		regURL := resolveRegistry()
		if regURL == "" {
			return fmt.Errorf("ERROR:P004: no registry configured")
		}
		rc := getRegistryClient()

		if publishDownloadURL == "" {
			return fmt.Errorf("ERROR:P006: --download-url is required (notary registry does not host files)")
		}

		req := registry.PublishRequest{
			Name:        m.Name,
			Version:     m.Version,
			Description: m.Description,
			Author:      m.Author,
			DownloadURL: publishDownloadURL,
			Tags:        publishTags,
			Scope:       publishScope,
			Visibility:  publishVisibility,
		}

		keyPath := publishKeyPath
		if keyPath == "" {
			keyPath = defaultSSHKeyPath()
		}

		if keyPath != "" {
			if _, err := os.Stat(keyPath); err == nil {
				return publishWithSSH(rc, keyPath, m, req)
			}
		}

		return publishWithToken(rc, m, req)
	},
}

func publishWithSSH(rc registry.Registry, keyPath string, m *archive.Manifest, req registry.PublishRequest) error {
	signer, err := auth.LoadPrivateKey(keyPath)
	if err != nil {
		return fmt.Errorf("ERROR:P008: load SSH key: %w", err)
	}

	fingerprint := auth.PublicKeyFingerprint(signer)

	manifestJSON, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("ERROR:P009: marshal manifest: %w", err)
	}

	signature, err := auth.SignManifest(signer, manifestJSON)
	if err != nil {
		return fmt.Errorf("ERROR:P010: sign manifest: %w", err)
	}

	if err := rc.PublishSSH(fingerprint, signature, req); err != nil {
		return fmt.Errorf("ERROR:P007: publish: %w", err)
	}

	fmt.Printf("Published %s v%s (ssh:%s)\n", m.Name, m.Version, fingerprint[:20])
	return nil
}

func publishWithToken(rc registry.Registry, m *archive.Manifest, req registry.PublishRequest) error {
	token := os.Getenv("CPM_REGISTRY_TOKEN")
	if token == "" {
		return fmt.Errorf("ERROR:P005: no SSH key found and CPM_REGISTRY_TOKEN not set")
	}

	fmt.Fprintln(os.Stderr, "Warning: using CPM_REGISTRY_TOKEN (deprecated, use SSH key auth)")

	manifestJSON, _ := json.Marshal(m)
	req.SHA256 = ""
	_ = manifestJSON

	if err := rc.Publish(token, req); err != nil {
		return fmt.Errorf("ERROR:P007: publish: %w", err)
	}

	fmt.Printf("Published %s v%s (token)\n", m.Name, m.Version)
	return nil
}

func init() {
	fs := publishCmd.Flags()
	fs.StringVar(&publishDownloadURL, "download-url", "", "Canonical download URL for the .cgp archive")
	fs.StringSliceVar(&publishTags, "tag", nil, "Tags for the package (repeatable)")
	fs.StringVar(&publishScope, "scope", "", "Package scope (e.g. username, org)")
	fs.StringVar(&publishVisibility, "visibility", "public", "Package visibility (public, private)")
	fs.StringVar(&publishKeyPath, "key", "", "SSH private key path (default: ~/.ssh/id_ed25519)")
	rootCmd.AddCommand(publishCmd)
}
