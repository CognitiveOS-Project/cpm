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
			return fmt.Errorf("ERROR:V001: open: %w", err)
		}
		defer f.Close()

		gzr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("ERROR:V002: invalid gzip: %w", err)
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
				return fmt.Errorf("ERROR:V003: invalid tar: %w", err)
			}

			name := filepath.Clean(hdr.Name)
			if name == "cognitive.json" {
				foundManifest = true
				if err := json.NewDecoder(tr).Decode(manifest); err != nil {
					return fmt.Errorf("ERROR:V004: invalid cognitive.json: %w", err)
				}
			}
			referencedFiles[name] = true
		}

		if !foundManifest {
			return fmt.Errorf("ERROR:V005: cognitive.json not found in archive")
		}

		doc := map[string]interface{}{
			"name":        manifest.Name,
			"version":     manifest.Version,
			"description": manifest.Description,
		}
		if err := schema.Validate(doc); err != nil {
			return fmt.Errorf("ERROR:V006: schema violation: %w", err)
		}

		if manifest.Runtime != nil {
			if manifest.Runtime.SystemPrompt != "" {
				if !referencedFiles[manifest.Runtime.SystemPrompt] {
					return fmt.Errorf("ERROR:V007: missing file: %s", manifest.Runtime.SystemPrompt)
				}
			}
			for _, srv := range manifest.Runtime.MCPServers {
				cmdPath := srv.Command
				if !filepath.IsAbs(cmdPath) {
					cmdPath = filepath.Join("tools", cmdPath)
				}
				if !referencedFiles[cmdPath] {
					return fmt.Errorf("ERROR:V008: missing MCP server binary: %s", srv.Command)
				}
			}
		}

		if manifest.Brain != nil {
			if manifest.Brain.Adapter != "" {
				if !referencedFiles[manifest.Brain.Adapter] {
					return fmt.Errorf("ERROR:V009: missing adapter file: %s", manifest.Brain.Adapter)
				}
			}
		}

		if manifest.Dependencies != nil {
			for depName := range manifest.Dependencies {
				depDir := filepath.Join("deps", depName)
				if !referencedFiles[depDir] && !referencedFiles[depDir+"/"] {
					fmt.Printf("  Note: dependency %s not bundled in archive (will be resolved at install)\n", depName)
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
