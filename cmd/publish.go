package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/auth"
	"github.com/CognitiveOS-Project/cpm/internal/registry"
	"github.com/spf13/cobra"
)

const maxCGPSize = 32 << 20 // 32 MB Cloud Run request limit

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
	Long: `Publish a .cgp archive to the configured registry.

Two publish modes are supported:

  Official (no --download-url):
    Uploads the .cgp to the registry, which creates a GitHub Release and
    stores the package. Best for packages under 32 MB.

  Notary proxy (--download-url required):
    Registers metadata only. The .cgp must be hosted externally (GitHub
    Releases, personal server, etc.). Consumers install via UPR:
      cpm install ghr:owner/repo@tag

For packages larger than 32 MB (e.g. with model weights), host the .cgp
on GitHub Releases and publish metadata via --download-url, or let
consumers install directly:
  cpm install ghr:owner/repo@tag

Authenticates using SSH key signing. Falls back to CPM_REGISTRY_TOKEN
(deprecated) if no SSH key is found.

Examples:
  cpm publish ./my-patch-1.0.0.cgp
  cpm publish ./my-patch-1.0.0.cgp --key ~/.ssh/my_key
  cpm publish ./my-patch-1.0.0.cgp --download-url https://github.com/.../my-patch-1.0.0.cgp
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

		keyPath := publishKeyPath
		if keyPath == "" {
			keyPath = defaultSSHKeyPath()
		}

		if publishDownloadURL == "" {
			return publishOfficial(rc, keyPath, path, m)
		}

		manifestJSON, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("ERROR:P009: marshal manifest: %w", err)
		}

		cgpData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("ERROR:P013: read %s: %w", path, err)
		}
		hash := sha256.Sum256(cgpData)

		req := registry.PublishRequest{
			Name:        m.Name,
			Version:     m.Version,
			Description: m.Description,
			Author:      m.Author,
			DownloadURL: publishDownloadURL,
			SHA256:      hex.EncodeToString(hash[:]),
			Tags:        publishTags,
			Scope:       publishScope,
			Visibility:  publishVisibility,
			Manifest:    manifestJSON,
		}

		if keyPath != "" {
			if _, err := os.Stat(keyPath); err == nil {
				return publishWithSSH(rc, keyPath, m, req)
			}
		}

		return publishWithToken(rc, m, req)
	},
}

func publishOfficial(rc registry.Registry, keyPath, cgpPath string, m *archive.Manifest) error {
	info, err := os.Stat(cgpPath)
	if err != nil {
		return fmt.Errorf("ERROR:P011: stat %s: %w", cgpPath, err)
	}
	if info.Size() > maxCGPSize {
		mb := info.Size() >> 20
		return fmt.Errorf(`ERROR:P012: .cgp file is %d MB, exceeds 32 MB registry server hosting limit.

For large packages, host the .cgp on GitHub Releases and publish metadata:
  cpm publish %s --download-url https://github.com/owner/repo/releases/download/tag/package.cgp

Or let consumers install directly via UPR:
  cpm install ghr:owner/repo@tag`, mb, cgpPath)
	}

	cgpData, err := os.ReadFile(cgpPath)
	if err != nil {
		return fmt.Errorf("ERROR:P013: read %s: %w", cgpPath, err)
	}

	if keyPath == "" {
		return fmt.Errorf("ERROR:P005: no SSH key found (official publish requires SSH auth)")
	}
	if _, err := os.Stat(keyPath); err != nil {
		return fmt.Errorf("ERROR:P005: SSH key not found: %s", keyPath)
	}

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

	req := registry.PublishRequest{
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
		Author:      m.Author,
		Tags:        publishTags,
		Scope:       publishScope,
		Visibility:  publishVisibility,
		Manifest:    manifestJSON,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("ERROR:P009: marshal request: %w", err)
	}

	if err := rc.PublishOfficial(fingerprint, signature, req, reqJSON, cgpData); err != nil {
		return fmt.Errorf("ERROR:P007: publish: %w", err)
	}

	fmt.Printf("Published %s v%s (official, ssh:%s)\n", m.Name, m.Version, fingerprint[:20])
	return nil
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

	if err := rc.Publish(token, req); err != nil {
		return fmt.Errorf("ERROR:P007: publish: %w", err)
	}

	fmt.Printf("Published %s v%s (token)\n", m.Name, m.Version)
	return nil
}

func init() {
	fs := publishCmd.Flags()
	fs.StringVar(&publishDownloadURL, "download-url", "", "Canonical download URL for notary proxy mode")
	fs.StringSliceVar(&publishTags, "tag", nil, "Tags for the package (repeatable)")
	fs.StringVar(&publishScope, "scope", "", "Package scope (e.g. username, org)")
	fs.StringVar(&publishVisibility, "visibility", "public", "Package visibility (public, private)")
	fs.StringVar(&publishKeyPath, "key", "", "SSH private key path (default: ~/.ssh/id_ed25519)")
	rootCmd.AddCommand(publishCmd)
}
