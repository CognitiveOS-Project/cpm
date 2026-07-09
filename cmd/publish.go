package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/registry"
	"github.com/spf13/cobra"
)

var publishDownloadURL string
var publishTags []string
var publishScope string
var publishVisibility string

var publishCmd = &cobra.Command{
	Use:   "publish <path>",
	Short: "Publish a .cgp patch to the registry",
	Long: `Publish a .cgp archive to the configured registry notary.

The SHA-256 checksum is computed locally and sent with the metadata.
The registry stores the notary record (checksum + download URL) and
redirects clients to the canonical download URL.

Examples:
  cpm publish ./my-patch-1.0.0.cgp --download-url https://github.com/.../my-patch-1.0.0.cgp
  cpm publish ./my-patch-1.0.0.cgp --tag vision --download-url https://...
  cpm publish ./my-patch-1.0.0.cgp --scope myorg --visibility private --download-url https://...`,
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

		_, _ = f.Seek(0, 0)
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			return fmt.Errorf("ERROR:P003: checksum: %w", err)
		}
		checksum := hex.EncodeToString(hasher.Sum(nil))

		regURL := resolveRegistry()
		if regURL == "" {
			return fmt.Errorf("ERROR:P004: no registry configured")
		}
		rc := registry.New(regURL)

		token := os.Getenv("CPM_REGISTRY_TOKEN")
		if token == "" {
			return fmt.Errorf("ERROR:P005: CPM_REGISTRY_TOKEN environment variable not set")
		}

		if publishDownloadURL == "" {
			return fmt.Errorf("ERROR:P006: --download-url is required (notary registry does not host files)")
		}

		req := registry.PublishRequest{
			Name:        m.Name,
			Version:     m.Version,
			Description: m.Description,
			Author:      m.Author,
			DownloadURL: publishDownloadURL,
			SHA256:      checksum,
			Tags:        publishTags,
			Scope:       publishScope,
			Visibility:  publishVisibility,
		}
		if err := rc.Publish(token, req); err != nil {
			return fmt.Errorf("ERROR:P007: publish: %w", err)
		}

		fmt.Printf("✓ Published %s v%s (sha256=%s)\n", m.Name, m.Version, checksum)
		return nil
	},
}

func init() {
	fs := publishCmd.Flags()
	fs.StringVar(&publishDownloadURL, "download-url", "", "Canonical download URL for the .cgp archive")
	fs.StringSliceVar(&publishTags, "tag", nil, "Tags for the package (repeatable)")
	fs.StringVar(&publishScope, "scope", "", "Package scope (e.g. username, org)")
	fs.StringVar(&publishVisibility, "visibility", "public", "Package visibility (public, private)")
	rootCmd.AddCommand(publishCmd)
}
