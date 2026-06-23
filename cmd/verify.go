package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/schema"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <path>",
	Short: "Verify a .cgp archive integrity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer f.Close()

		// Check tar.gz format
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("invalid gzip: %w", err)
		}
		defer gzr.Close()

		tr := tar.NewReader(gzr)
		foundManifest := false
		referencedFiles := map[string]bool{}
		manifest := &archive.Manifest{}

		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("invalid tar: %w", err)
			}

			name := filepath.Clean(hdr.Name)
		if name == "cognitive.json" {
				foundManifest = true
				if err := json.NewDecoder(tr).Decode(manifest); err != nil {
					return fmt.Errorf("invalid cognitive.json: %w", err)
				}
			}
			referencedFiles[name] = true
		}

		if !foundManifest {
			return fmt.Errorf("cognitive.json not found in archive")
		}

		// Validate schema
		doc := map[string]interface{}{
			"name":        manifest.Name,
			"version":     manifest.Version,
			"description": manifest.Description,
		}
		if err := schema.Validate(doc); err != nil {
			return fmt.Errorf("schema violation: %w", err)
		}

		// Check referenced files exist
		if manifest.Runtime != nil {
			if manifest.Runtime.SystemPrompt != "" {
				if !referencedFiles[manifest.Runtime.SystemPrompt] {
					return fmt.Errorf("missing file: %s", manifest.Runtime.SystemPrompt)
				}
			}
			for _, srv := range manifest.Runtime.MCPServers {
				cmdPath := srv.Command
				if !filepath.IsAbs(cmdPath) {
					cmdPath = filepath.Join("tools", cmdPath)
				}
				if !referencedFiles[cmdPath] {
					return fmt.Errorf("missing MCP server binary: %s", srv.Command)
				}
			}
		}

		fmt.Printf("✓ %s is valid (%s v%s)\n", filepath.Base(path), manifest.Name, manifest.Version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
